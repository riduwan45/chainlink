package log

import (
	"database/sql"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/smartcontractkit/chainlink/core/store/models"
)

//go:generate mockery --name ORM --output ./mocks/ --case=underscore --structname ORM --filename orm.go

type ORM interface {
	WasBroadcastConsumed(blockHash common.Hash, logIndex uint, jobID interface{}) (bool, error)
	MarkBroadcastConsumed(blockHash common.Hash, logIndex uint, jobID interface{}) error
}

type orm struct {
	db *gorm.DB
}

var _ ORM = (*orm)(nil)

func NewORM(db *gorm.DB) *orm {
	return &orm{db}
}

func (o *orm) WasBroadcastConsumed(blockHash common.Hash, logIndex uint, jobID interface{}) (consumed bool, err error) {
	var jobIDName string
	switch v := jobID.(type) {
	case models.JobID:
		jobIDName = "job_id"
	case int32:
		jobIDName = "job_id_v2"
	default:
		panic(fmt.Sprintf("unrecognised type for jobID: %T", v))
	}

	q := `
        SELECT consumed FROM log_broadcasts
        WHERE block_hash = ?
        AND log_index = ?
        AND %s = ?
    `

	args := []interface{}{
		blockHash,
		logIndex,
		jobID,
	}

	stmt := fmt.Sprintf(q, jobIDName)
	err = o.db.Raw(stmt, args...).Row().Scan(&consumed)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return consumed, err
}

func (o *orm) MarkBroadcastConsumed(blockHash common.Hash, logIndex uint, jobID interface{}) error {
	var jobIDName string
	switch v := jobID.(type) {
	case models.JobID:
		jobIDName = "job_id"
	case int32:
		jobIDName = "job_id_v2"
	default:
		panic(fmt.Sprintf("unrecognised type for jobID: %T", v))
	}

	query := o.db.Exec(fmt.Sprintf(`
        INSERT INTO log_broadcasts (block_hash, log_index, %s, created_at, consumed) VALUES (?, ?, ?, NOW(), true)
    `, jobIDName), blockHash, logIndex, jobID)
	if query.Error != nil {
		return errors.Wrap(query.Error, "while marking log broadcast as consumed")
	} else if query.RowsAffected == 0 {
		return errors.Errorf("cannot mark log broadcast as consumed: does not exist")
	}
	return nil
}
