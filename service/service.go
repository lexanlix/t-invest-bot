package service

import (
	"context"
	"time"

	"t-api/entity"
	"t-api/internal/log"

	"github.com/pkg/errors"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

type Repository interface {
	GetUserAccounts() ([]*pb.Account, error)
	GetLastOperations(req *investgo.GetOperationsRequest) ([]*pb.Operation, error)
	GetInstrumentByFigi(id string) (*pb.Instrument, error)
	GetInstrumentByUid(uid string) (*pb.Instrument, error)
}

type TgRepo interface {
	SendOperations(operations []entity.Operation) error
}

//nolint:containedctx
type Service struct {
	cancelCtx             context.Context
	logger                log.Logger
	investRepo            Repository
	tgRepo                TgRepo
	accountId             string
	checkOperationsPeriod time.Duration
	doneChan              chan struct{}
}

func NewService(cancelCtx context.Context, logger log.Logger, investRepo Repository, tgRepo TgRepo,
	operationsTimeout time.Duration, accountId string) *Service {
	return &Service{
		cancelCtx:             cancelCtx,
		logger:                logger,
		investRepo:            investRepo,
		tgRepo:                tgRepo,
		accountId:             accountId,
		checkOperationsPeriod: operationsTimeout,
	}
}

func (s *Service) StartTracing() {
	s.doneChan = make(chan struct{})
	go s.run()
	s.logger.Info(s.cancelCtx, "started tracing account", log.String("accountId", s.accountId))
}

func (s *Service) PauseTracing() {
	close(s.doneChan)
	s.logger.Info(s.cancelCtx, "paused tracing account", log.String("accountId", s.accountId))
}

func (s *Service) GetAccounts() ([]*pb.Account, error) {
	accounts, err := s.investRepo.GetUserAccounts()
	if err != nil {
		return nil, errors.WithMessage(err, "get user accounts")
	}

	return accounts, nil
}

func (s *Service) FetchOperations() error {
	req := &investgo.GetOperationsRequest{
		AccountId: s.accountId,
		State:     pb.OperationState_OPERATION_STATE_UNSPECIFIED,
		From:      time.Now().Add(-s.checkOperationsPeriod),
		To:        time.Now(),
	}

	operations, err := s.investRepo.GetLastOperations(req)
	if err != nil {
		return errors.WithMessage(err, "get last operations")
	}

	if len(operations) == 0 {
		s.logger.Debug(s.cancelCtx, "operations not found", log.String("accountId", s.accountId))
		return nil
	}

	formatted, err := s.formatOperations(operations)
	if err != nil {
		return errors.WithMessage(err, "format operations")
	}

	err = s.tgRepo.SendOperations(formatted)
	if err != nil {
		return errors.WithMessage(err, "send operations")
	}

	return nil
}

func (s *Service) run() {
	t := time.NewTicker(s.checkOperationsPeriod)
	defer t.Stop()

	for {
		select {
		case <-s.cancelCtx.Done():
			close(s.doneChan)
			return
		case _, isOpen := <-s.doneChan:
			if !isOpen {
				return
			}
		case <-t.C:
			err := s.FetchOperations()
			if err != nil {
				s.logger.Error(s.cancelCtx, "send operations", log.Any("error", err))
			}
		}
	}
}

func (s *Service) formatOperations(operations []*pb.Operation) ([]entity.Operation, error) {
	result := make([]entity.Operation, 0)

	for _, op := range operations {
		var (
			instrument *pb.Instrument
			err        error
		)

		hasInstrument := true
		switch {
		case len(op.GetFigi()) != 0:
			instrument, err = s.investRepo.GetInstrumentByFigi(op.GetFigi())
			if err != nil {
				return nil, errors.WithMessagef(err, "get instrument by figi '%s', op.Type '%s'", op.GetFigi(), op.GetType())
			}

		case len(op.GetInstrumentUid()) != 0:
			instrument, err = s.investRepo.GetInstrumentByUid(op.GetInstrumentUid())
			if err != nil {
				return nil, errors.WithMessagef(err, "get instrument by figi '%s', op.Type '%s'", op.GetFigi(), op.GetType())
			}
		default:
			hasInstrument = false
		}

		result = append(result, entity.Operation{
			Operation:     op,
			HasInstrument: hasInstrument,
			Instrument:    instrument,
		})
	}

	return result, nil
}
