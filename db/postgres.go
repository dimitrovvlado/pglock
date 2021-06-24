package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/lopezator/migrator"
)

//DB is a global DB connection designed to be accessed by everyone
var DB *sql.DB

var DefaultTTL float64 = 300

//A list of DDL migrations
var migrations = []interface{}{
	&migrator.Migration{
		Name: "Init Vault PostgreSQL tables",
		Func: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS LOCK (
					id                    SERIAL PRIMARY KEY NOT NULL,
					profile_id            VARCHAR (255) UNIQUE NOT NULL,
					device_id             VARCHAR (255) NOT NULL,
					created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT Now()
				);
			`); err != nil {
				return err
			}
			return nil
		},
	},
}

//Lock is the data object which correponds to the LOCK table
type Lock struct {
	ID        int
	ProfileID string
	DeviceID  string
	CreatedAt time.Time
}

func Init(ds string, attempts int) error {
	var err error
	//Init database connection
	DB, err = sql.Open("postgres", ds)
	if err != nil {
		return err
	}

	for i := 1; i <= attempts; i++ {
		fmt.Printf("Waiting for database to start (attempt %d of %d)\n", i, attempts)
		_, err := DB.Exec("SELECT 1")
		if err == nil {
			fmt.Printf("Database is up\n")
			break
		}
		fmt.Printf("Error connecting: %v\n", err)
		if i >= attempts {
			return fmt.Errorf("Database connection failed after %d attempts", i)
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func AttemptLock(profileId, deviceId string) (bool, error) {
	//TODO use context properly
	ctx := context.Background()
	var lock Lock
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}

	//select the row for update effectively locking it
	row := tx.QueryRowContext(ctx, `SELECT * FROM LOCK WHERE profile_id = $1 FOR UPDATE NOWAIT`, profileId)

	//try to read the result set
	err = row.Scan(&lock.ID, &lock.ProfileID, &lock.DeviceID, &lock.CreatedAt)
	if err != nil {
		//either the row does not exist, or someone has already acquired the lock
		//try inserting to distiguish the 2 cases
		_, err = tx.ExecContext(ctx, `
			INSERT INTO LOCK (profile_id, device_id, created_at)
			VALUES ($1, $2, $3)`, profileId, deviceId, time.Now())
		if err != nil {
			//rollback and return, someone else is accessing this data row
			tx.Rollback()
			return false, err
		}
		tx.Commit()
		return true, nil
	}

	//If we're trying to acquire a lock for the same device or the
	//lease has expired regardless of the device, update the expiration
	sinceRefresh := time.Now().Sub(lock.CreatedAt).Seconds()
	if lock.DeviceID == deviceId || sinceRefresh > DefaultTTL {
		_, err = tx.ExecContext(ctx, `
			UPDATE LOCK SET device_id = $1, created_at = $2 WHERE profile_id = $3`,
			lock.DeviceID, time.Now(), lock.ProfileID)
		if err == nil {
			//successfully aquired lock
			tx.Commit()
			return true, nil
		}
	}
	//Lock hasn't expired, but is held by another device
	tx.Rollback()
	return false, nil
}

//Releases the lock for a given profileID
func ReleaseLock(profileId string) (int, error) {
	res, err := DB.Exec(`
		UPDATE LOCK SET created_at = $1 WHERE profile_id = $2`,
		time.Now().Add(-10*time.Hour), profileId)
	if err != nil {
		return 0, err
	}
	rows, _ := res.RowsAffected()
	return int(rows), nil
}

//Truncate is used for testing purposes only
func Truncate() error {
	_, err := DB.Exec(`TRUNCATE TABLE LOCK`)
	return err
}

//MigrateDatabase runs the schema migrations. It's meant to be used
//first thing before the server is marked healthy
func MigrateDatabase() error {
	m, err := migrator.New(migrator.Migrations(migrations...))
	if err != nil {
		return err
	}
	return m.Migrate(DB)
}
