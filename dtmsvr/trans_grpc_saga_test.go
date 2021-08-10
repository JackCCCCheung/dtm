package dtmsvr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yedf/dtm/dtmcli"
	"github.com/yedf/dtm/dtmgrpc"
	"github.com/yedf/dtm/examples"
)

func TestGrpcSaga(t *testing.T) {
	sagaGrpcNormal(t)
	sagaGrpcCommittedPending(t)
	sagaGrpcRollback(t)
}

func sagaGrpcNormal(t *testing.T) {
	saga := genSagaGrpc("gid-sagaGrpcNormal", false, false)
	saga.Submit()
	WaitTransProcessed(saga.Gid)
	assert.Equal(t, []string{"prepared", "succeed", "prepared", "succeed"}, getBranchesStatus(saga.Gid))
	assert.Equal(t, "succeed", getTransStatus(saga.Gid))
	transQuery(t, saga.Gid)
}

func sagaGrpcCommittedPending(t *testing.T) {
	saga := genSagaGrpc("gid-committedPendingGrpc", false, false)
	examples.MainSwitch.TransOutResult.SetOnce("PENDING")
	saga.Submit()
	WaitTransProcessed(saga.Gid)
	assert.Equal(t, []string{"prepared", "prepared", "prepared", "prepared"}, getBranchesStatus(saga.Gid))
	CronTransOnce(60 * time.Second)
	assert.Equal(t, []string{"prepared", "succeed", "prepared", "succeed"}, getBranchesStatus(saga.Gid))
	assert.Equal(t, "succeed", getTransStatus(saga.Gid))
}

func sagaGrpcRollback(t *testing.T) {
	saga := genSagaGrpc("gid-rollbackSaga2Grpc", false, true)
	examples.MainSwitch.TransOutRevertResult.SetOnce("PENDING")
	saga.Submit()
	WaitTransProcessed(saga.Gid)
	assert.Equal(t, "aborting", getTransStatus(saga.Gid))
	CronTransOnce(60 * time.Second)
	assert.Equal(t, "failed", getTransStatus(saga.Gid))
	assert.Equal(t, []string{"succeed", "succeed", "succeed", "failed"}, getBranchesStatus(saga.Gid))
}

func genSagaGrpc(gid string, outFailed bool, inFailed bool) *dtmgrpc.SagaGrpc {
	dtmcli.Logf("beginning a grpc saga test ---------------- %s", gid)
	saga := dtmgrpc.NewSaga(examples.DtmGrpcServer, gid)
	req := dtmcli.MustMarshal(examples.GenTransReq(30, outFailed, inFailed))
	saga.Add(examples.BusiGrpc+"/examples.Busi/TransOut", examples.BusiGrpc+"/examples.Busi/TransOutRevert", req)
	saga.Add(examples.BusiGrpc+"/examples.Busi/TransIn", examples.BusiGrpc+"/examples.Busi/TransInRevert", req)
	return saga
}
