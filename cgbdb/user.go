package cgbdb

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/heroiclabs/nakama-common/runtime"
)

func UpdateUserPlayingInMatch(ctx context.Context, logger runtime.Logger, db *sql.DB, userIdD string, matchId string) error {
	query := `UPDATE
					users AS u
				SET
					metadata
						= u.metadata
						|| jsonb_build_object('playing_in_match','` + matchId + `' )
				WHERE	
					id = $1;`
	_, err := db.ExecContext(ctx, query, userIdD)
	if err != nil {
		logger.WithField("err", err).Error("db.ExecContext match update error.")
	}
	return err
}

func UpdateUsersPlayingInMatch(ctx context.Context, logger runtime.Logger, db *sql.DB, userIds []string, matchId string) error {
	if len(userIds) == 0 {
		return nil
	}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(
		`UPDATE
					users AS u
				SET
					metadata
						= u.metadata
						|| jsonb_build_object('playing_in_match','` + matchId + `' )
				WHERE	
					id IN ( `)
	args := make([]any, 0)
	lenUids := len(userIds)
	for i, uid := range userIds {
		queryBuilder.WriteString("$")
		idx := i + 1
		queryBuilder.WriteString(strconv.Itoa(idx))
		if idx < lenUids {
			queryBuilder.WriteString(",")
		}
		args = append(args, uid)
	}
	queryBuilder.WriteString(" );")
	query := queryBuilder.String()
	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		logger.WithField("err", err).Error("db.ExecContext match update error.")
	}
	return err
}
