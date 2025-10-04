package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/db"
	"gitlab.kalliopedata.io/genai-apps/pinazu-core/internal/service"
)

const (
	// RequestTimeOut is the timeout for the request to the tool server
	RequestTimeOut time.Duration = 60 * time.Second // 1 minutes for long-running AI operations

	// RequestRetries is the retry time for the request to the tool server
	RequestRetries uint8 = 3

	// RequestDelay is the delay between retries to the tool server
	RequestDelay time.Duration = 2 * time.Second
)

// executeStandaloneTool send post request to each of the tool server
func (ts *ToolService) executeStandaloneTool(standaloneToolsToExecute []service.StandaloneToolRequestEventMessage, header *service.EventHeaders, meta *service.EventMetadata) {
	if len(standaloneToolsToExecute) == 0 {
		return
	}

	for _, t := range standaloneToolsToExecute {
		go func(ctx context.Context, t service.StandaloneToolRequestEventMessage) {
			var resp *http.Response
			var err error

			c, cancel := context.WithTimeout(ctx, RequestTimeOut)
			defer cancel()
			b, err := json.Marshal(t.ToolInput)
			if err != nil {
				ts.log.Error("Failed to marshal tool input", "error", err)
				return
			}

			client := &http.Client{}
			var success bool

			for i := range RequestRetries {
				// Create a new request for each retry attempt to avoid body reader exhaustion
				req, err := http.NewRequestWithContext(c, "POST", t.ToolURL, bytes.NewReader(b))
				if err != nil {
					ts.log.Error("Failed to create new tool standalone request", "error", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")
				if t.ToolAPIKey != nil {
					req.Header.Set("Authorization", "Bearer "+*t.ToolAPIKey)
				}

				resp, err = client.Do(req)
				if resp != nil {
					defer resp.Body.Close()
				}

				if err == nil && resp.StatusCode < 500 {
					body, readErr := io.ReadAll(resp.Body)
					if readErr != nil {
						ts.log.Error("Failed to read response body", "error", readErr)
						// Send error event for read body failure
						errorContent, _ := db.NewJsonRaw(map[string]any{"error": readErr.Error()})
						event := service.NewEvent(&service.ToolGatherEventMessage{
							ToolRunId:  t.ToolRunId,
							Content:    errorContent,
							ResultType: db.ResultMessageTypeText,
							IsError:    true,
						}, header, &service.EventMetadata{
							TraceID:   meta.TraceID,
							Timestamp: time.Now(),
						})
						if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
							ts.log.Error("failed to publish error to tool gather event", "error", publishErr)
						}
						success = true
						break
					}

					// Check if HTTP status indicates error
					if resp.StatusCode >= 400 {
						// Send error event for HTTP errors
						errorContent, _ := db.NewJsonRaw(map[string]any{"error": fmt.Sprintf("HTTP error %d: %s", resp.StatusCode, string(body))})
						event := service.NewEvent(&service.ToolGatherEventMessage{
							ToolRunId:  t.ToolRunId,
							Content:    errorContent,
							ResultType: db.ResultMessageTypeText,
							IsError:    true,
						}, header, &service.EventMetadata{
							TraceID:   meta.TraceID,
							Timestamp: time.Now(),
						})
						if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
							ts.log.Error("failed to publish error to tool gather event", "error", publishErr)
						}
					} else {
						// Success case - use NewEvent
						// Parse the JSON response first, then convert to JsonRaw
						var parsedResponse any
						if err := json.Unmarshal(body, &parsedResponse); err != nil {
							ts.log.Error("Failed to parse response JSON", "error", err)
							// Send error event for JSON parsing failure
							errorContent, _ := db.NewJsonRaw(map[string]any{"error": err.Error()})
							event := service.NewEvent(&service.ToolGatherEventMessage{
								ToolRunId:  t.ToolRunId,
								Content:    errorContent,
								ResultType: db.ResultMessageTypeText,
								IsError:    true,
							}, header, &service.EventMetadata{
								TraceID:   meta.TraceID,
								Timestamp: time.Now(),
							})
							if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
								ts.log.Error("failed to publish error to tool gather event", "error", publishErr)
							}
							success = true
							break
						}

						content, err := db.NewJsonRaw(parsedResponse)
						if err != nil {
							ts.log.Error("Failed to convert parsed response to JsonRaw", "error", err)
							// Send error event for JSON conversion failure
							errorContent, _ := db.NewJsonRaw(map[string]any{"error": err.Error()})
							event := service.NewEvent(&service.ToolGatherEventMessage{
								ToolRunId:  t.ToolRunId,
								Content:    errorContent,
								ResultType: db.ResultMessageTypeText,
								IsError:    true,
							}, header, &service.EventMetadata{
								TraceID:   meta.TraceID,
								Timestamp: time.Now(),
							})
							if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
								ts.log.Error("failed to publish error to tool gather event", "error", publishErr)
							}
						} else {
							// Success case - use NewEvent only for non-errors
							event := service.NewEvent(&service.ToolGatherEventMessage{
								ToolRunId:  t.ToolRunId,
								Content:    content,
								ResultType: db.ResultMessageTypeText,
								IsError:    false,
							}, header, &service.EventMetadata{
								TraceID:   meta.TraceID,
								Timestamp: time.Now(),
							})
							if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
								ts.log.Error("failed to publish result to tool gather event", "error", publishErr)
							}
						}
					}
					success = true
					break
				}

				if err != nil {
					ts.log.Error("Failed to send post request to tool standalone", "name", t.ToolName, "error", err, "attempt", i+1)
				}

				if i < RequestRetries-1 {
					time.Sleep(RequestDelay)
				}
			}

			if !success {
				errorMsg := "Tool execution failed after all retries"
				if err != nil {
					errorMsg = err.Error()
				} else if resp != nil && resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					errorMsg = string(body)
					ts.log.Error("Standalone tool response with 500", "name", t.ToolName, "error", errorMsg)
				}

				// Create error tool gather event with tool run ID
				errorContent, _ := db.NewJsonRaw(map[string]any{"error": errorMsg})
				event := service.NewEvent(&service.ToolGatherEventMessage{
					ToolRunId:  t.ToolRunId,
					Content:    errorContent,
					ResultType: db.ResultMessageTypeText,
					IsError:    true,
				}, header, &service.EventMetadata{
					TraceID:   meta.TraceID,
					Timestamp: time.Now(),
				})
				if publishErr := event.Publish(ts.s.GetNATS()); publishErr != nil {
					ts.log.Error("failed to publish error to tool gather event", "error", publishErr)
				}
			}
		}(ts.ctx, t)
	}

	ts.log.Info("Send concurrent request to standalone tool", "tool_name", standaloneToolsToExecute)
}
