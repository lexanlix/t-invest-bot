package entity

import pb "github.com/russianinvestments/invest-api-go-sdk/proto"

type Operation struct {
	*pb.Operation

	HasInstrument bool
	Instrument    *pb.Instrument
}
