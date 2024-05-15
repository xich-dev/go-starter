package model

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/logger"
	"github.com/xich-dev/go-starter/pkg/model/querier"
	"github.com/xich-dev/go-starter/pkg/utils"
)

var log = logger.NewLogAgent("model")

var (
	ErrAlreadyInTransaction = errors.New("already in transaction")
)

type ModelInterface interface {
	querier.Querier
	RunTransaction(ctx context.Context, f func(model ModelInterface) error) error
	InTransaction() bool
}

type Model struct {
	querier.Querier
	beginTx       func(ctx context.Context) (pgx.Tx, error)
	p             *pgxpool.Pool
	inTransaction bool
}

func (m *Model) InTransaction() bool {
	return m.inTransaction
}

func (m *Model) RunTransaction(ctx context.Context, f func(model ModelInterface) error) error {
	tx, err := m.beginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := f(
		&Model{
			Querier: querier.New(tx),
			beginTx: func(ctx context.Context) (pgx.Tx, error) {
				return nil, ErrAlreadyInTransaction
			},
			inTransaction: true,
		},
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (m *Model) dataInit() error {
	log.Info("running data init on database")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stmt := "INSERT INTO access_rules (name) VALUES "
	for i, rule := range AllRules {
		stmt += fmt.Sprintf("('%s')%s", rule, utils.IfElse(i == len(AllRules)-1, "", ","))
	}
	stmt += "ON CONFLICT DO NOTHING"

	_, err := m.p.Exec(ctx, stmt)
	return err
}

func NewModel(cfg *config.Config) (ModelInterface, error) {
	url := fmt.Sprintf("%s:%s@%s:%d/%s?connect_timeout=%d&timezone=Asia/Shanghai", cfg.Pg.User, cfg.Pg.Password, cfg.Pg.Host, cfg.Pg.Port, cfg.Pg.Db, 15)
	dsn := fmt.Sprintf("postgres://%s", url)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse pgxpool config: %s", dsn)
	}

	var (
		retryLimit = 10
		retry      = 0
	)

	var p *pgxpool.Pool

	for {
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()

		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			log.Warnf("failed to init pgxpool: %s", err.Error())
			if retry >= retryLimit {
				return nil, errors.Wrapf(err, "failed to init pgxpool: %s", dsn)
			}
			continue
		}

		p = pool

		if err := pool.Ping(ctx); err != nil {
			log.Warnf("failed to ping database: %s", err.Error())
			if retry >= retryLimit {
				return nil, errors.Wrap(err, "failed to ping db")
			}
		} else {
			break
		}
		retry++
		time.Sleep(3 * time.Second)
	}

	m, err := migrate.New(fmt.Sprintf("file://%s", cfg.Pg.Migration), fmt.Sprintf("pgx5://%s", url))
	if err != nil {
		return nil, errors.Wrap(err, "failed to init migrate")
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return nil, errors.Wrap(err, "failed to migrate up")
		}
	}

	model := &Model{Querier: querier.New(p), beginTx: p.Begin, p: p}
	if err := model.dataInit(); err != nil {
		return nil, errors.Wrap(err, "failed to init data")
	}
	log.Info("model init success")

	return model, nil
}
