package archivist

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
)

// Archive old build in a dedicated table
func Archive(interval, nHoursKeepsBuild int) {

	// If this goroutine exits, then it's a crash
	defer log.Fatalf("Goroutine of archivist.Archive exited - Exit CDS Engine")

	for {
		time.Sleep(time.Duration(interval) * time.Second)
		db := database.DB()
		if db != nil {
			buildIDs, err := pipeline.LoadBuildIDsToArchive(db, nHoursKeepsBuild)
			if err != nil {
				log.Warning("Archive> Cannot load buildIDs to archive: %s\n", err)
				continue
			}
			log.Notice("Archive> Loaded %d pipeline_build older than %d hours\n", len(buildIDs), nHoursKeepsBuild)

			for _, id := range buildIDs {
				// Take your time
				time.Sleep(500 * time.Millisecond)
				lockAndArchiveBuild(db, id)
			}
		}
	}
}

// This function sole purpose is to handler the transaction
// and query pipeline_build with a FOR UPDATE to lock it in transaction
// WARNING: defer ONLY runs when FUNCTION returns, not when context exit
func lockAndArchiveBuild(db *sql.DB, id int64) {
	tx, err := db.Begin()
	if err != nil {
		log.Warning("lockAndArchiveBuild> Cannot start transaction: %s\n", err)
		return
	}
	defer tx.Rollback()

	err = ArchiveBuild(tx, id)
	if err != nil {
		pqerr, ok := err.(*pq.Error)
		// Cannot get lock (FOR UPDATE NOWAIT), someone else is on it
		if ok && pqerr.Code == "55P03" {
			log.Info("lockAndArchiveBuild> %s", err)
			return
		}
		if strings.Contains(err.Error(), "could not obtain lock on row") {
			// Yes I known, it's bad to check error string, I failed to find the definition in lib/pq
			log.Notice("lockAndArchiveBuild> %s", err)
			// Close tx and wait 10s
			tx.Rollback()
			time.Sleep(10 * time.Second)
			return
		}
		log.Warning("lockAndArchiveBuild> Cannot archive build %d: %s\n", id, err)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Warning("LockAndArchiveBuild> Cannot commit transaction: %s\n", err)
	}
}

// ArchiveBuild archives given pipeline build
func ArchiveBuild(db database.QueryExecuter, id int64) error {
	query := `SELECT id FROM pipeline_build WHERE id = $1 FOR UPDATE NOWAIT`
	err := db.QueryRow(query, id).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		log.Info("ArchiveBuild> Error while selecting PipelineBuild %s", err)
		return err
	}
	log.Debug("ArchiveBuild> Archiving PipelineBuild %d\n", id)
	completeBuild, err := pipeline.LoadCompletePipelineBuildToArchive(db, id)
	if err != nil {
		log.Warning("ArchiveBuild> Error while loading PipelineBuild %d : %s\n", id, err)
		return fmt.Errorf("cannot load complete build information: %s", err)
	}
	if err := pipeline.SavePipelineBuildHistory(db, completeBuild); err != nil {
		return fmt.Errorf("cannot archive build: %s", err)
	}

	if err := pipeline.DeleteBuild(db, id); err != nil {
		return fmt.Errorf("cannot delete build: %s", err)
	}

	return nil
}
