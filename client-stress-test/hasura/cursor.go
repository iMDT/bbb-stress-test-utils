package hasura

import (
	"math/rand"
	"time"

	"bbb-stress-test/common"
)

const (
	// Bounding box for simulated cursor positions inside the whiteboard.
	cursorMinX = 0.0
	cursorMaxX = 1420.0
	cursorMinY = 0.0
	cursorMaxY = 800.0

	// Per-tick movement: each axis moves by a random delta in [-step, +step]
	// to keep the path continuous and human-looking.
	cursorMaxStep = 60.0

	// Minimum interval between two cursor positions during an active burst.
	cursorMinIntervalMs = 100
	// Extra random jitter added on top of the minimum, in milliseconds.
	cursorMaxJitterMs = 150

	// Probability that any given tick is a "pause" (cursor held still, no
	// mutation sent because the server already has the last position).
	cursorPauseProbability = 0.15
	// Pause duration range, in milliseconds.
	cursorPauseMinMs = 300
	cursorPauseMaxMs = 2000

	// Active burst duration: a random number of seconds in [min, max].
	cursorActiveMinSecs = 2
	cursorActiveMaxSecs = 10
	// Idle gap between bursts.
	cursorIdleMinSecs = 1
	cursorIdleMaxSecs = 12
)

// RunCursorPublisher repeatedly simulates a user moving the whiteboard cursor.
// While user.WhiteboardWriteAccess is true, it alternates between active bursts
// (sending positions every ~100ms, each one a small step from the previous so
// the path looks like a real moving cursor) and idle gaps (sending nothing).
// Within a burst the cursor may also pause: when it does, no mutation is sent
// because the server already holds the last position. Each burst ends with an
// "off-screen" (-1, -1) cursor before going idle. Exits when whiteboard write
// access is revoked or the WebSocket connection is closed.
func RunCursorPublisher(user *common.User) {
	defer func() { user.CursorPublishingActive = false }()

	user.Logger.Infoln("Cursor publisher started")

	// Persist position across bursts so the cursor reappears near where it left off.
	x := cursorMinX + rand.Float64()*(cursorMaxX-cursorMinX)
	y := cursorMinY + rand.Float64()*(cursorMaxY-cursorMinY)

	for {
		if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
			user.Logger.Infoln("Cursor publisher stopping")
			return
		}

		if user.CurrentPageId == "" {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		whiteboardId := user.CurrentPageId

		burstSecs := cursorActiveMinSecs + rand.Intn(cursorActiveMaxSecs-cursorActiveMinSecs+1)
		burstUntil := time.Now().Add(time.Duration(burstSecs) * time.Second)

		for time.Now().Before(burstUntil) {
			if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
				break
			}
			if user.CurrentPageId != whiteboardId {
				// Page changed mid-burst: restart loop so the next burst uses the new page.
				break
			}

			if rand.Float64() < cursorPauseProbability {
				// Hold still — the server already has this position, so don't resend.
				pauseMs := cursorPauseMinMs + rand.Intn(cursorPauseMaxMs-cursorPauseMinMs+1)
				time.Sleep(time.Duration(pauseMs) * time.Millisecond)
				continue
			}

			x = clamp(x+(rand.Float64()*2-1)*cursorMaxStep, cursorMinX, cursorMaxX)
			y = clamp(y+(rand.Float64()*2-1)*cursorMaxStep, cursorMinY, cursorMaxY)
			SendPresentationPublishCursor(user, whiteboardId, x, y)
			time.Sleep(time.Duration(cursorMinIntervalMs+rand.Intn(cursorMaxJitterMs+1)) * time.Millisecond)
		}

		// Always send the off-screen marker before going idle.
		if !user.WsConnectionClosed {
			SendPresentationPublishCursor(user, whiteboardId, -1, -1)
		}

		if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
			user.Logger.Infoln("Cursor publisher stopping")
			return
		}

		idleSecs := cursorIdleMinSecs + rand.Intn(cursorIdleMaxSecs-cursorIdleMinSecs+1)
		time.Sleep(time.Duration(idleSecs) * time.Second)
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// SendPresentationPublishCursor sends a single PresentationPublishCursor
// mutation with the given whiteboardId and percent coordinates.
func SendPresentationPublishCursor(user *common.User, whiteboardId string, xPercent, yPercent float64) {
	if user.WsConnectionClosed {
		return
	}
	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{
			"whiteboardId": whiteboardId,
			"xPercent":     xPercent,
			"yPercent":     yPercent,
		},
		"PresentationPublishCursor",
		`mutation PresentationPublishCursor($whiteboardId: String!, $xPercent: Float!, $yPercent: Float!) {
			presentationPublishCursor(
				whiteboardId: $whiteboardId
				xPercent: $xPercent
				yPercent: $yPercent
			)
		}`)
}
