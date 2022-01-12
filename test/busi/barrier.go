/*
 * Copyright (c) 2021 yedf. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package busi

import (
	"context"
	"database/sql"

	"github.com/dtm-labs/dtm/dtmcli"
	"github.com/dtm-labs/dtm/dtmutil"
	"github.com/gin-gonic/gin"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	setupFuncs["BarrierSetup"] = func(app *gin.Engine) {
		app.POST(BusiAPI+"/SagaBTransIn", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			barrier := MustBarrierFromGin(c)
			return barrier.Call(txGet(), func(tx *sql.Tx) error {
				return SagaAdjustBalance(tx, TransInUID, reqFrom(c).Amount, reqFrom(c).TransInResult)
			})
		}))
		app.POST(BusiAPI+"/SagaBTransInCompensate", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			barrier := MustBarrierFromGin(c)
			return barrier.Call(txGet(), func(tx *sql.Tx) error {
				return SagaAdjustBalance(tx, TransInUID, -reqFrom(c).Amount, "")
			})
		}))
		app.POST(BusiAPI+"/SagaBTransOut", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			barrier := MustBarrierFromGin(c)
			return barrier.Call(txGet(), func(tx *sql.Tx) error {
				return SagaAdjustBalance(tx, TransOutUID, -reqFrom(c).Amount, reqFrom(c).TransOutResult)
			})
		}))
		app.POST(BusiAPI+"/SagaBTransOutCompensate", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			barrier := MustBarrierFromGin(c)
			return barrier.Call(txGet(), func(tx *sql.Tx) error {
				return SagaAdjustBalance(tx, TransOutUID, reqFrom(c).Amount, "")
			})
		}))
		app.POST(BusiAPI+"/SagaBTransOutGorm", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			req := reqFrom(c)
			barrier := MustBarrierFromGin(c)
			tx := dbGet().DB.Begin()
			return barrier.Call(tx.Statement.ConnPool.(*sql.Tx), func(tx1 *sql.Tx) error {
				return tx.Exec("update dtm_busi.user_account set balance = balance + ? where user_id = ?", -req.Amount, TransOutUID).Error
			})
		}))

		app.POST(BusiAPI+"/TccBTransInTry", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			req := reqFrom(c)
			if req.TransInResult != "" {
				return dtmcli.String2DtmError(req.TransInResult)
			}
			return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
				return tccAdjustTrading(tx, TransInUID, req.Amount)
			})
		}))
		app.POST(BusiAPI+"/TccBTransInConfirm", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
				return tccAdjustBalance(tx, TransInUID, reqFrom(c).Amount)
			})
		}))
		app.POST(BusiAPI+"/TccBTransInCancel", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
				return tccAdjustTrading(tx, TransInUID, -reqFrom(c).Amount)
			})
		}))
		app.POST(BusiAPI+"/TccBTransOutTry", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			req := reqFrom(c)
			if req.TransOutResult != "" {
				return dtmcli.String2DtmError(req.TransOutResult)
			}
			return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
				return tccAdjustTrading(tx, TransOutUID, -req.Amount)
			})
		}))
		app.POST(BusiAPI+"/TccBTransOutConfirm", dtmutil.WrapHandler2(func(c *gin.Context) interface{} {
			return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
				return tccAdjustBalance(tx, TransOutUID, -reqFrom(c).Amount)
			})
		}))
		app.POST(BusiAPI+"/TccBTransOutCancel", dtmutil.WrapHandler2(TccBarrierTransOutCancel))
	}
}

// TccBarrierTransOutCancel will be use in test
func TccBarrierTransOutCancel(c *gin.Context) interface{} {
	return MustBarrierFromGin(c).Call(txGet(), func(tx *sql.Tx) error {
		return tccAdjustTrading(tx, TransOutUID, reqFrom(c).Amount)
	})
}

func (s *busiServer) TransInBSaga(ctx context.Context, in *BusiReq) (*emptypb.Empty, error) {
	barrier := MustBarrierFromGrpc(ctx)
	return &emptypb.Empty{}, barrier.Call(txGet(), func(tx *sql.Tx) error {
		return sagaGrpcAdjustBalance(tx, TransInUID, in.Amount, in.TransInResult)
	})
}

func (s *busiServer) TransOutBSaga(ctx context.Context, in *BusiReq) (*emptypb.Empty, error) {
	barrier := MustBarrierFromGrpc(ctx)
	return &emptypb.Empty{}, barrier.Call(txGet(), func(tx *sql.Tx) error {
		return sagaGrpcAdjustBalance(tx, TransOutUID, -in.Amount, in.TransOutResult)
	})
}

func (s *busiServer) TransInRevertBSaga(ctx context.Context, in *BusiReq) (*emptypb.Empty, error) {
	barrier := MustBarrierFromGrpc(ctx)
	return &emptypb.Empty{}, barrier.Call(txGet(), func(tx *sql.Tx) error {
		return sagaGrpcAdjustBalance(tx, TransInUID, -in.Amount, "")
	})
}

func (s *busiServer) TransOutRevertBSaga(ctx context.Context, in *BusiReq) (*emptypb.Empty, error) {
	barrier := MustBarrierFromGrpc(ctx)
	return &emptypb.Empty{}, barrier.Call(txGet(), func(tx *sql.Tx) error {
		return sagaGrpcAdjustBalance(tx, TransOutUID, in.Amount, "")
	})
}

func (s *busiServer) QueryPreparedB(ctx context.Context, in *BusiReq) (*emptypb.Empty, error) {
	barrier := MustBarrierFromGrpc(ctx)
	return &emptypb.Empty{}, barrier.QueryPrepared(dbGet().ToSQLDB())
}