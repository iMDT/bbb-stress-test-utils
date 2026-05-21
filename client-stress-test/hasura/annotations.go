package hasura

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"bbb-stress-test/common"
)

const (
	// Per-user cap on live annotations. Once reached, the publisher only moves
	// or deletes existing shapes — no new creations — matching the server-side
	// limit of 300 annotations per pageId.
	maxAnnotationsPerUser = 300

	// Annotation cadence is intentionally slower than the cursor's: real users
	// draw individual shapes much less frequently than they move the mouse.
	annotationActiveMinSecs = 4
	annotationActiveMaxSecs = 15
	annotationIdleMinSecs   = 5
	annotationIdleMaxSecs   = 25

	annotationMinIntervalMs = 600
	annotationMaxJitterMs   = 1400

	// Alphabet matching the tldraw shape-id format (`shape:` + 21 chars).
	shapeIdAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

	// Text "typing" cadence: each character append is a separate mutation that
	// bumps the shape's meta.version, mirroring real-user behavior captured in
	// the annotations_subscription example.
	textInitialW       = 8.0
	textWPerChar       = 11.5
	textTypingMinMs    = 150
	textTypingJitterMs = 200
)

var annotationShapeTypes = []string{"draw", "geo", "text", "arrow", "highlight"}

var sampleTextWords = []string{
	"hello", "world", "stress", "test", "BBB",
	"note", "draft", "todo", "wow", "agenda",
}

// Style pools used when creating or restyling annotations. Each shape type
// applies only the props that make sense for it (see styleForType).
var (
	annotationColors = []string{
		"black", "blue", "green", "grey",
		"light-blue", "light-green", "light-red", "light-violet",
		"orange", "red", "violet", "yellow",
	}
	annotationSizes  = []string{"s", "m", "l", "xl"}
	annotationFills  = []string{"none", "semi", "solid", "pattern"}
	annotationDashes = []string{"draw", "solid", "dashed", "dotted"}
	annotationFonts  = []string{"draw", "sans", "serif", "mono"}
	annotationGeos   = []string{
		"rectangle", "ellipse", "triangle", "diamond",
		"pentagon", "hexagon", "octagon", "rhombus",
		"oval", "star", "x-box", "check-box",
	}
)

func randomFromList(list []string) string { return list[rand.Intn(len(list))] }

// RunAnnotationsPublisher repeatedly simulates a user submitting whiteboard
// annotations. Like the cursor publisher, it gates on WhiteboardWriteAccess
// and alternates active bursts with idle gaps; each emitted annotation
// originates at the current shared cursor position (user.CursorX/CursorY).
// Once the per-user cap is reached, only move and delete actions are issued.
func RunAnnotationsPublisher(user *common.User) {
	defer func() { user.AnnotationPublishingActive = false }()

	user.Logger.Infoln("Annotations publisher started")

	if user.Annotations == nil {
		user.Annotations = make(map[string]map[string]interface{})
	}

	// If the cursor publisher isn't running, seed a position so annotations
	// don't all pile up at (0, 0).
	if user.CursorX == 0 && user.CursorY == 0 {
		user.CursorX = cursorMinX + rand.Float64()*(cursorMaxX-cursorMinX)
		user.CursorY = cursorMinY + rand.Float64()*(cursorMaxY-cursorMinY)
	}

	for {
		if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
			user.Logger.Infoln("Annotations publisher stopping")
			return
		}

		if user.CurrentPageId == "" {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		pageId := user.CurrentPageId

		burstSecs := annotationActiveMinSecs + rand.Intn(annotationActiveMaxSecs-annotationActiveMinSecs+1)
		burstUntil := time.Now().Add(time.Duration(burstSecs) * time.Second)

		for time.Now().Before(burstUntil) {
			if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
				break
			}
			if user.CurrentPageId != pageId {
				// Page changed mid-burst; new bursts use the new page.
				break
			}

			pickAndSendAnnotationAction(user, pageId)

			time.Sleep(time.Duration(annotationMinIntervalMs+rand.Intn(annotationMaxJitterMs+1)) * time.Millisecond)
		}

		if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
			user.Logger.Infoln("Annotations publisher stopping")
			return
		}

		idleSecs := annotationIdleMinSecs + rand.Intn(annotationIdleMaxSecs-annotationIdleMinSecs+1)
		time.Sleep(time.Duration(idleSecs) * time.Second)
	}
}

func pickAndSendAnnotationAction(user *common.User, pageId string) {
	atCap := len(user.AnnotationsOrder) >= maxAnnotationsPerUser
	hasExisting := len(user.AnnotationsOrder) > 0

	if atCap {
		// No new creations past the cap — only move, restyle, or delete.
		roll := rand.Intn(100)
		switch {
		case roll < 50:
			moveRandomAnnotation(user, pageId)
		case roll < 85:
			restyleRandomAnnotation(user, pageId)
		default:
			deleteRandomAnnotation(user, pageId)
		}
		return
	}

	if !hasExisting {
		createNewAnnotation(user, pageId)
		return
	}

	roll := rand.Intn(100)
	switch {
	case roll < 70:
		createNewAnnotation(user, pageId)
	case roll < 82:
		moveRandomAnnotation(user, pageId)
	case roll < 95:
		restyleRandomAnnotation(user, pageId)
	default:
		deleteRandomAnnotation(user, pageId)
	}
}

func createNewAnnotation(user *common.User, pageId string) {
	shapeType := annotationShapeTypes[rand.Intn(len(annotationShapeTypes))]
	id := randomShapeId()
	user.AnnotationIndex++
	idx := "a" + strconv.Itoa(user.AnnotationIndex)

	presentationId, parentId := splitPageId(pageId)

	info := map[string]interface{}{
		"id":       id,
		"x":        user.CursorX,
		"y":        user.CursorY,
		"rotation": 0,
		"isLocked": false,
		"opacity":  1,
		"meta": map[string]interface{}{
			"version":        1,
			"createdBy":      user.UserId,
			"updatedBy":      user.UserId,
			"presentationId": presentationId,
		},
		"type":        shapeType,
		"props":       buildAnnotationProps(shapeType),
		"parentId":    parentId,
		"index":       idx,
		"typeName":    "shape",
		"isModerator": user.IsModerator,
	}

	user.Annotations[id] = info
	user.AnnotationsOrder = append(user.AnnotationsOrder, id)

	sendPresAnnotationSubmit(user, pageId, id, info)

	if shapeType == "text" {
		typeOutText(user, pageId, id, info)
	}
}

// typeOutText simulates a user typing into a freshly-created text shape: it
// picks a random word and appends one character per mutation, each bumping
// meta.version and the shape's width. Aborts mid-word if write access is
// revoked, the connection drops, or the active page changes.
func typeOutText(user *common.User, pageId, id string, info map[string]interface{}) {
	word := sampleTextWords[rand.Intn(len(sampleTextWords))]

	for i := 1; i <= len(word); i++ {
		if !user.WhiteboardWriteAccess || user.WsConnectionClosed {
			return
		}
		if user.CurrentPageId != pageId {
			return
		}

		time.Sleep(time.Duration(textTypingMinMs+rand.Intn(textTypingJitterMs+1)) * time.Millisecond)

		if props, ok := info["props"].(map[string]interface{}); ok {
			props["text"] = word[:i]
			props["w"] = textInitialW + float64(i)*textWPerChar
		}
		if meta, ok := info["meta"].(map[string]interface{}); ok {
			v, _ := meta["version"].(int)
			meta["version"] = v + 1
			meta["synced"] = true
			meta["updatedBy"] = user.UserId
		}

		sendPresAnnotationSubmit(user, pageId, id, info)
	}
}

func moveRandomAnnotation(user *common.User, pageId string) {
	if len(user.AnnotationsOrder) == 0 {
		return
	}
	id := user.AnnotationsOrder[rand.Intn(len(user.AnnotationsOrder))]
	info, ok := user.Annotations[id]
	if !ok {
		return
	}

	info["x"] = user.CursorX
	info["y"] = user.CursorY

	if meta, ok := info["meta"].(map[string]interface{}); ok {
		v, _ := meta["version"].(int)
		meta["version"] = v + 1
		meta["synced"] = true
		meta["updatedBy"] = user.UserId
	}

	sendPresAnnotationSubmit(user, pageId, id, info)
}

// restyleRandomAnnotation picks an existing shape, randomizes its style props
// (color/size/fill/dash/font, as applicable to the shape's type), bumps the
// meta.version, and re-sends. Mirrors a user clicking a shape and changing
// styles in the toolbar.
func restyleRandomAnnotation(user *common.User, pageId string) {
	if len(user.AnnotationsOrder) == 0 {
		return
	}
	id := user.AnnotationsOrder[rand.Intn(len(user.AnnotationsOrder))]
	info, ok := user.Annotations[id]
	if !ok {
		return
	}

	shapeType, _ := info["type"].(string)
	props, ok := info["props"].(map[string]interface{})
	if !ok {
		return
	}

	applyRandomStyle(shapeType, props)

	if meta, ok := info["meta"].(map[string]interface{}); ok {
		v, _ := meta["version"].(int)
		meta["version"] = v + 1
		meta["synced"] = true
		meta["updatedBy"] = user.UserId
	}

	sendPresAnnotationSubmit(user, pageId, id, info)
}

// applyRandomStyle mutates props with fresh random style values appropriate to
// shapeType. Geometry/text/segments fields are left untouched.
func applyRandomStyle(shapeType string, props map[string]interface{}) {
	color := randomFromList(annotationColors)
	size := randomFromList(annotationSizes)

	switch shapeType {
	case "draw":
		props["color"] = color
		props["size"] = size
		props["fill"] = randomFromList(annotationFills)
		props["dash"] = randomFromList(annotationDashes)
	case "highlight":
		props["color"] = color
		props["size"] = size
	case "geo":
		props["color"] = color
		props["labelColor"] = color
		props["size"] = size
		props["fill"] = randomFromList(annotationFills)
		props["dash"] = randomFromList(annotationDashes)
		props["font"] = randomFromList(annotationFonts)
	case "text":
		props["color"] = color
		props["size"] = size
		props["font"] = randomFromList(annotationFonts)
	case "arrow":
		props["color"] = color
		props["labelColor"] = color
		props["size"] = size
		props["fill"] = randomFromList(annotationFills)
		props["dash"] = randomFromList(annotationDashes)
		props["font"] = randomFromList(annotationFonts)
	}
}

func deleteRandomAnnotation(user *common.User, pageId string) {
	if len(user.AnnotationsOrder) == 0 {
		return
	}
	i := rand.Intn(len(user.AnnotationsOrder))
	id := user.AnnotationsOrder[i]

	user.AnnotationsOrder = append(user.AnnotationsOrder[:i], user.AnnotationsOrder[i+1:]...)
	delete(user.Annotations, id)

	sendPresAnnotationDelete(user, pageId, []string{id})
}

func sendPresAnnotationSubmit(user *common.User, pageId, id string, info map[string]interface{}) {
	if user.WsConnectionClosed {
		return
	}
	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{
			"pageId": pageId,
			"annotations": []map[string]interface{}{
				{
					"id":             id,
					"annotationInfo": info,
					"wbId":           pageId,
					"userId":         user.UserId,
				},
			},
		},
		"PresAnnotationSubmit",
		`mutation PresAnnotationSubmit($pageId: String!, $annotations: json!) {
			presAnnotationSubmit(pageId: $pageId, annotations: $annotations)
		}`)
}

func sendPresAnnotationDelete(user *common.User, pageId string, ids []string) {
	if user.WsConnectionClosed {
		return
	}
	SendGenericGraphqlMessage(
		user,
		GetCurrMessageId(user),
		map[string]interface{}{
			"pageId":         pageId,
			"annotationsIds": ids,
		},
		"PresAnnotationDelete",
		`mutation PresAnnotationDelete($pageId: String!, $annotationsIds: [String]!) {
			presAnnotationDelete(pageId: $pageId, annotationsIds: $annotationsIds)
		}`)
}

func randomShapeId() string {
	b := make([]byte, 21)
	for i := range b {
		b[i] = shapeIdAlphabet[rand.Intn(len(shapeIdAlphabet))]
	}
	return "shape:" + string(b)
}

// splitPageId extracts presentationId and parentId from a pageId of the form
// "<presentationId>/<num>", e.g. "abc-123/4" → ("abc-123", "page:4").
func splitPageId(pageId string) (presentationId, parentId string) {
	idx := strings.LastIndex(pageId, "/")
	if idx < 0 {
		return pageId, "page:1"
	}
	return pageId[:idx], "page:" + pageId[idx+1:]
}

func buildAnnotationProps(shapeType string) map[string]interface{} {
	color := randomFromList(annotationColors)
	size := randomFromList(annotationSizes)
	fill := randomFromList(annotationFills)
	dash := randomFromList(annotationDashes)
	font := randomFromList(annotationFonts)

	switch shapeType {
	case "draw":
		return map[string]interface{}{
			"segments":   []map[string]interface{}{{"type": "free", "points": randomDrawPoints()}},
			"color":      color,
			"fill":       fill,
			"dash":       dash,
			"size":       size,
			"isComplete": true,
			"isClosed":   false,
			"isPen":      false,
		}
	case "highlight":
		return map[string]interface{}{
			"segments":   []map[string]interface{}{{"type": "free", "points": randomDrawPoints()}},
			"color":      color,
			"size":       size,
			"isComplete": true,
			"isPen":      false,
		}
	case "geo":
		w := 80.0 + rand.Float64()*220.0
		h := 60.0 + rand.Float64()*180.0
		return map[string]interface{}{
			"w":             w,
			"h":             h,
			"geo":           randomFromList(annotationGeos),
			"color":         color,
			"labelColor":    color,
			"fill":          fill,
			"dash":          dash,
			"size":          size,
			"font":          font,
			"text":          "",
			"align":         "start",
			"verticalAlign": "start",
			"growY":         0,
			"url":           "",
		}
	case "text":
		return map[string]interface{}{
			"color":    color,
			"size":     size,
			"w":        8,
			"text":     "",
			"font":     font,
			"align":    "start",
			"autoSize": true,
			"scale":    1,
		}
	case "arrow":
		endX := (rand.Float64()*2 - 1) * 300
		endY := (rand.Float64()*2 - 1) * 200
		return map[string]interface{}{
			"dash":           dash,
			"size":           size,
			"fill":           fill,
			"color":          color,
			"labelColor":     color,
			"bend":           0,
			"start":          map[string]interface{}{"type": "point", "x": 0, "y": 0},
			"end":            map[string]interface{}{"type": "point", "x": endX, "y": endY},
			"arrowheadStart": "none",
			"arrowheadEnd":   "arrow",
			"text":           "",
			"font":           font,
		}
	default:
		return map[string]interface{}{}
	}
}

func randomDrawPoints() []map[string]interface{} {
	n := 5 + rand.Intn(15)
	points := make([]map[string]interface{}, n)
	x, y := 0.0, 0.0
	for i := 0; i < n; i++ {
		x += (rand.Float64()*2 - 1) * 30
		y += (rand.Float64()*2 - 1) * 30
		points[i] = map[string]interface{}{
			"x": x,
			"y": y,
			"z": 0.5,
		}
	}
	return points
}
