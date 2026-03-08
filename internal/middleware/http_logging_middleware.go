package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"go-timekeeper/internal/logger"

	"github.com/gin-gonic/gin"
)

const maxLoggedBodyBytes = 4096

// HTTPLogging logs incoming requests and outgoing responses with request metadata.
func HTTPLogging(log *logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		requestID := requestIDFromContext(ctx)

		baseFields := buildRequestLogFields(ctx)
		baseFields["request_id"] = requestID
		baseFields["request_content_type"] = ctx.GetHeader("Content-Type")

		requestBody, requestBodyErr := readAndRestoreRequestBody(ctx.Request)
		if requestBodyErr != nil {
			baseFields["request_body_read_error"] = requestBodyErr.Error()
		} else {
			requestBodyText, requestBodyTruncated := normalizeBodyForLog(
				baseFields["request_content_type"].(string),
				requestBody,
			)
			baseFields["request_body"] = requestBodyText
			baseFields["request_body_truncated"] = requestBodyTruncated
		}

		respCapture := &bodyCaptureWriter{
			ResponseWriter: ctx.Writer,
			body:           &bytes.Buffer{},
		}
		ctx.Writer = respCapture

		log.WithFields(baseFields).Info("incoming_http_request")

		ctx.Next()

		latency := time.Since(start)
		if route := ctx.FullPath(); route != "" {
			baseFields["route"] = route
		}
		baseFields["status_code"] = ctx.Writer.Status()
		baseFields["response_code"] = ctx.Writer.Status()
		baseFields["latency_ms"] = latency.Milliseconds()
		baseFields["response_size_bytes"] = ctx.Writer.Size()
		baseFields["response_content_type"] = ctx.Writer.Header().Get("Content-Type")

		responseBodyText, responseBodyTruncated := normalizeBodyForLog(
			baseFields["response_content_type"].(string),
			respCapture.body.Bytes(),
		)
		baseFields["response_body"] = responseBodyText
		baseFields["response_body_truncated"] = responseBodyTruncated

		if len(ctx.Errors) > 0 {
			baseFields["errors"] = ctx.Errors.String()
		}

		entry := log.WithFields(baseFields)
		if ctx.Writer.Status() >= http.StatusInternalServerError {
			entry.Error(logger.LogMessageOutgoingHTTPResponse)
			return
		}
		if ctx.Writer.Status() >= http.StatusBadRequest {
			entry.Warn(logger.LogMessageOutgoingHTTPResponse)
			return
		}

		entry.Info(logger.LogMessageOutgoingHTTPResponse)
	}
}

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	if _, err := w.body.Write(b); err != nil {
		return 0, err
	}
	return w.ResponseWriter.Write(b)
}

func (w *bodyCaptureWriter) WriteString(s string) (int, error) {
	if _, err := w.body.WriteString(s); err != nil {
		return 0, err
	}
	return w.ResponseWriter.WriteString(s)
}

func readAndRestoreRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(payload))

	return payload, nil
}

func normalizeBodyForLog(contentType string, body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}

	if !isTextLikeContentType(contentType) {
		return "<omitted: non-text body>", false
	}

	raw := strings.TrimSpace(string(body))
	if len(raw) <= maxLoggedBodyBytes {
		return raw, false
	}

	return raw[:maxLoggedBodyBytes], true
}

func isTextLikeContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" {
		return true
	}

	return strings.HasPrefix(ct, "application/json") ||
		strings.HasPrefix(ct, "application/xml") ||
		strings.HasPrefix(ct, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(ct, "text/")
}

func buildRequestLogFields(ctx *gin.Context) logger.Fields {
	fields := logger.Fields{
		"method":     ctx.Request.Method,
		"path":       ctx.Request.URL.Path,
		"route":      ctx.FullPath(),
		"query":      ctx.Request.URL.RawQuery,
		"client_ip":  ctx.ClientIP(),
		"user_agent": ctx.Request.UserAgent(),
	}

	for _, param := range ctx.Params {
		key := strings.ToLower(param.Key)
		value := strings.TrimSpace(param.Value)
		if value == "" {
			continue
		}

		fields["param_"+key] = value

		switch key {
		case "id":
			fields["object_id"] = value
		case "sku":
			fields["sku"] = value
		case "quote_id":
			fields["quote_id"] = value
		case "order_id":
			fields["order_id"] = value
		}

		if strings.Contains(key, "reservation") && strings.Contains(key, "id") {
			fields["reservation_id"] = value
		}
	}

	return fields
}
