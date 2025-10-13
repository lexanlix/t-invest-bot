package repository

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Invest struct {
	cli *investgo.Client

	userRepo       *investgo.UsersServiceClient
	operationRepo  *investgo.OperationsServiceClient
	instrumentRepo *investgo.InstrumentsServiceClient
}

func NewInvest(ctx context.Context, config investgo.Config) (*Invest, error) {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)
	zapConfig.EncoderConfig.TimeKey = "time"

	log, err := zapConfig.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "build logger")
	}

	cli, err := investgo.NewClient(ctx, config, log.Sugar())
	if err != nil {
		return nil, errors.WithMessage(err, "create investgo client")
	}

	return &Invest{
		cli:            cli,
		userRepo:       cli.NewUsersServiceClient(),
		operationRepo:  cli.NewOperationsServiceClient(),
		instrumentRepo: cli.NewInstrumentsServiceClient(),
	}, nil
}

// GetUserAccounts Получение списка открытых и активных счетов
func (r Invest) GetUserAccounts() ([]*pb.Account, error) {
	status := pb.AccountStatus_ACCOUNT_STATUS_OPEN
	res, err := r.userRepo.GetAccounts(&status)
	if err != nil {
		return nil, errors.WithMessage(err, "get user accounts")
	}

	return res.GetAccounts(), nil
}

// GetLastOperations Получение списка последних операций счета
func (r Invest) GetLastOperations(req *investgo.GetOperationsRequest) ([]*pb.Operation, error) {
	res, err := r.operationRepo.GetOperations(req)
	if err != nil {
		return nil, errors.WithMessagef(err, "get last operations by account '%s'", req.AccountId)
	}

	return res.GetOperations(), nil
}

func (r Invest) GetInstrumentByFigi(id string) (*pb.Instrument, error) {
	res, err := r.instrumentRepo.InstrumentByFigi(id)
	if err != nil {
		return nil, errors.WithMessagef(err, "get instrument by figi '%s'", id)
	}

	return res.Instrument, nil
}

func (r Invest) GetInstrumentByUid(uid string) (*pb.Instrument, error) {
	res, err := r.instrumentRepo.InstrumentByUid(uid)
	if err != nil {
		return nil, errors.WithMessagef(err, "get instrument by uid '%s'", uid)
	}

	return res.Instrument, nil
}
