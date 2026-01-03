package server

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/btcsuite/websocket"
	"github.com/haileyok/cocoon/metrics"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleSyncSubscribeRepos(e echo.Context) error {
	ctx := e.Request().Context()
	logger := s.logger.With("component", "subscribe-repos-websocket")

	conn, err := websocket.Upgrade(e.Response().Writer, e.Request(), e.Response().Header(), 1<<10, 1<<10)
	if err != nil {
		logger.Error("unable to establish websocket with relay", "err", err)
		return err
	}

	ident := e.RealIP() + "-" + e.Request().UserAgent()
	logger = logger.With("ident", ident)
	logger.Info("new connection established")

	metrics.RelaysConnected.WithLabelValues(ident).Inc()
	defer func() {
		metrics.RelaysConnected.WithLabelValues(ident).Dec()
	}()

	// Parse cursor parameter for replay
	cursor := int64(0)
	if cursorStr := e.QueryParam("cursor"); cursorStr != "" {
		if parsed, err := strconv.ParseInt(cursorStr, 10, 64); err == nil {
			cursor = parsed
		}
	}

	// Helper to send event over websocket
	sendEvent := func(evt *events.XRPCStreamEvent) error {
		wc, err := conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return err
		}

		header := events.EventHeader{Op: events.EvtKindMessage}
		var obj util.CBOR

		switch {
		case evt.Error != nil:
			header.Op = events.EvtKindErrorFrame
			obj = evt.Error
		case evt.RepoCommit != nil:
			header.MsgType = "#commit"
			obj = evt.RepoCommit
		case evt.RepoIdentity != nil:
			header.MsgType = "#identity"
			obj = evt.RepoIdentity
		case evt.RepoAccount != nil:
			header.MsgType = "#account"
			obj = evt.RepoAccount
		case evt.RepoInfo != nil:
			header.MsgType = "#info"
			obj = evt.RepoInfo
		default:
			return nil // skip unknown events
		}

		if err := header.MarshalCBOR(wc); err != nil {
			return err
		}

		if err := obj.MarshalCBOR(wc); err != nil {
			return err
		}

		if err := wc.Close(); err != nil {
			return err
		}

		metrics.RelaySends.WithLabelValues(ident, header.MsgType).Inc()
		return nil
	}

	if cursor > 0 && s.dbPersister != nil {
		oldestSeq, err := s.dbPersister.GetOldestSeq(ctx)
		if err == nil && cursor < oldestSeq && oldestSeq > 0 {
			sendEvent(&events.XRPCStreamEvent{
				RepoInfo: &atproto.SyncSubscribeRepos_Info{
					Name:    "OutdatedCursor",
					Message: to.StringPtr("Cursor is older than available events"),
				},
			})
		}

		logger.Info("replaying events from cursor", "cursor", cursor)
		err = s.dbPersister.Playback(ctx, cursor, func(evt *events.XRPCStreamEvent) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return sendEvent(evt)
		})
		if err != nil {
			logger.Error("error during playback", "err", err)
		}
	}

	evts, cancel, err := s.evtman.Subscribe(ctx, ident, func(evt *events.XRPCStreamEvent) bool {
		return true
	}, nil)
	if err != nil {
		return err
	}
	defer cancel()

	for evt := range evts {
		func() {
			if ctx.Err() != nil {
				return
			}
			if err := sendEvent(evt); err != nil {
				logger.Error("error sending event", "err", err)
			}
		}()
	}

	// we should tell the relay to request a new crawl at this point if we got disconnected
	// use a new context since the old one might be cancelled at this point
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.requestCrawl(ctx); err != nil {
		logger.Error("error requesting crawls", "err", err)
	}

	return nil
}
