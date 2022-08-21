package cloudfront

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edwardofclt/cloudfront-emulator/internal/lambda"
	originrequest "github.com/edwardofclt/cloudfront-emulator/internal/origin-request"
	originresponse "github.com/edwardofclt/cloudfront-emulator/internal/origin-response"
	"github.com/edwardofclt/cloudfront-emulator/internal/origins"
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	viewerrequest "github.com/edwardofclt/cloudfront-emulator/internal/viewer-request"
	viewerresponse "github.com/edwardofclt/cloudfront-emulator/internal/viewer-response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Cloudfront struct {
	server      *http.Server
	handler     http.Handler
	pathToCerts string
	wg          *sync.WaitGroup
}

type CloudfrontConfig struct {
	Address       *string           `mapstructure:"address"`
	Port          *int              `mapstructure:"port"`
	OriginConfigs map[string]Origin `mapstructure:"origins"`
	Behaviors     []Behavior        `mapstructure:"behaviors"`
}

type Origin struct {
	Domain string
	Path   string
}

type EventType string

const (
	ViewerRequest  EventType = "viewer-request"
	OriginRequest  EventType = "origin-request"
	OriginResponse EventType = "origin-response"
	ViewerResponse EventType = "viewer-response"
)

var EventTypes []EventType = []EventType{
	ViewerRequest,
	OriginRequest,
	OriginResponse,
	ViewerResponse,
}

type Behavior struct {
	Path   string
	Origin string
	Events map[EventType]Event
}

type Event struct {
	Path    string
	Handler string
}

type EventResponse struct {
}

func NewV2(config *types.CloudfrontConfig) *CfServer {
	if config == nil {
		logrus.Fatal("CloudfrontConfig was nil")
	}

	addr := "localhost"
	if config.Address != nil {
		addr = *config.Address
	}

	port := 443
	if config.Port != nil {
		port = *config.Port
	}

	eventHandlers := map[types.EventType]types.CloudfrontEvent{
		types.ViewerRequest:  viewerrequest.New(),
		types.OriginRequest:  originrequest.New(),
		types.OriginResponse: originresponse.New(),
		types.ViewerResponse: viewerresponse.New(),
	}

	cf := &CfServer{
		Server: &http.Server{
			Addr: fmt.Sprintf("%s:%d", addr, port),
		},
		Handler:       generateRoutesV2(config, eventHandlers),
		Wg:            &sync.WaitGroup{},
		EventHandlers: eventHandlers,
	}

	if port == 443 {
		cf.PathToCerts = generateCertsForSSL(addr)
	}

	startServerV2(cf)

	return cf
}

func New(config *CloudfrontConfig) *Cloudfront {
	if config == nil {
		logrus.Fatal("CloudfrontConfig was nil")
	}

	addr := "localhost"
	if config.Address != nil {
		addr = *config.Address
	}

	port := 443
	if config.Port != nil {
		port = *config.Port
	}

	cf := &Cloudfront{
		server: &http.Server{
			Addr: fmt.Sprintf("%s:%d", addr, port),
		},
		handler: generateRoutes(config),
		wg:      &sync.WaitGroup{},
	}

	cf.pathToCerts = generateCertsForSSL(addr)
	startServer(cf)

	return cf
}

func makeOriginRequest(r *http.Request, requestPayload RequestPayload, origin Origin) (*CfResponse, error) {
	requestURL := filepath.Clean(fmt.Sprintf("%s/%s/%s", origin.Domain, origin.Path, r.URL.Path))
	fullURL := fmt.Sprintf("%s://%s", strings.ToLower(strings.Split(r.Proto, "/")[0]), requestURL)

	originRequest, _ := http.NewRequest(r.Method, fullURL, r.Body)

	for _, value := range *requestPayload.Records[0].Cf.Request.Headers {
		originRequest.Header.Add(value[0].Key, value[0].Value)
	}

	originResponse, err := http.DefaultClient.Do(originRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching the origin")
	}

	originResponseData, err := ioutil.ReadAll(originResponse.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing origin response")
	}

	statusCode := strconv.Itoa(originResponse.StatusCode)

	finalResponse := &CfResponse{
		Body:    originResponseData,
		Status:  &statusCode,
		Headers: &CfHeaderArray{},
	}

	for key, value := range originResponse.Header {
		header := *finalResponse.Headers
		header[key] = []CfHeader{
			{
				Key:   key,
				Value: value[0],
			},
		}
		finalResponse.Headers = &header
	}

	return finalResponse, nil
}

func makeOriginRequestV2(r *http.Request, requestPayload types.RequestPayload, origin types.Origin) (*types.CfResponse, error) {
	requestURL := filepath.Clean(fmt.Sprintf("%s/%s/%s", origin.Domain, origin.Path, r.URL.Path))
	fullURL := fmt.Sprintf("%s://%s", strings.ToLower(strings.Split(r.Proto, "/")[0]), requestURL)

	originRequest, _ := http.NewRequest(r.Method, fullURL, r.Body)

	for _, value := range *requestPayload.Records[0].Cf.Request.Headers {
		originRequest.Header.Add(value[0].Key, value[0].Value)
	}

	originResponse, err := http.DefaultClient.Do(originRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching the origin")
	}

	originResponseData, err := ioutil.ReadAll(originResponse.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing origin response")
	}

	statusCode := strconv.Itoa(originResponse.StatusCode)

	finalResponse := &types.CfResponse{
		Body:    originResponseData,
		Status:  &statusCode,
		Headers: &types.CfHeaderArray{},
	}

	for key, value := range originResponse.Header {
		header := *finalResponse.Headers
		header[key] = []types.CfHeader{
			{
				Key:   key,
				Value: value[0],
			},
		}
		finalResponse.Headers = &header
	}

	return finalResponse, nil
}

func generateRoutes(config *CloudfrontConfig) *chi.Mux {
	handlers := chi.NewRouter()

	sort.Slice(config.Behaviors, func(i, j int) bool {
		return config.Behaviors[i].Path > config.Behaviors[j].Path
	})

	for _, behaviorValue := range config.Behaviors {
		// make a copy since behaviorValue is a pointer in the slice
		behavior := behaviorValue
		origin, ok := config.OriginConfigs[behavior.Origin]
		if !ok {
			logrus.Fatalf("bad configuration: behavior uses undefined origin: %s", behavior.Origin)
		}

		handlers.HandleFunc(behavior.Path, func(w http.ResponseWriter, r *http.Request) {
			// listen for callback content
			var callbackContent []byte
			callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error

				callbackContent, err = ioutil.ReadAll(r.Body)
				if err != nil {
					sendErrorResponse(w, "failed to parse callback content", err.Error())
					return
				}
			}))
			defer callback.Close()

			var finalResponse *CfResponse
			requestId := uuid.New()
			w.Header().Add("x-lambda-emulator-requestId", requestId.String())

			requestPayload := generateRequestBody(requestId, ViewerRequest, r)

			for _, event := range EventTypes {
				// this needs to run regardless before we run origin-response
				if event == OriginResponse {
					var err error
					finalResponse, err = makeOriginRequest(r, requestPayload, origin)
					if err != nil {
						sendErrorResponse(w, "failed to makeOriginRequest", err.Error())
						return
					}

					requestPayload.Records[0].Cf.Response = finalResponse
				}

				// if we don't have a behavior to act on, continue to the next event type
				eventHandler, ok := behavior.Events[event]
				if !ok {
					continue
				}

				requestPayload.Records[0].Cf.Config.EventType = event

				payload, err := requestPayload.EncodeJSON()
				if err != nil {
					logrus.WithError(err).Fatal("something went wrong marshaling the request")
					sendErrorResponse(w, "failed to generate status", err.Error())
					return
				}

				resp, err := executeLambda(payload, eventHandler, callback)
				if err != nil {
					sendErrorResponse(w, fmt.Sprintf(`failed to execute the lambda
			%s`, resp), err.Error())
					return
				}

				// TODO: validate response headeres and data aren't immutable according to AWS docs
				// - [x] Viewer Request
				// - [x] Origin Request
				// - [ ] Origin Response
				// - [ ] Viewer Response

				if event == ViewerRequest || event == OriginRequest {
					respData, err := parseRequestData(string(callbackContent))
					if err != nil {
						sendErrorResponse(w, "failed to parse request response data", err.Error())
						return
					}

					requestPayload = mergeRequestResponseWithRequestPayload(requestPayload, respData)

					// if respData.Headers != nil {
					// 	for key, val := range *respData.Headers {
					// 		headers := *requestPayload.Records[0].Cf.Request.Headers
					// 		headers[key] = val
					// 		requestPayload.Records[0].Cf.Request.Headers = &headers
					// 	}
					// }

					if event == ViewerRequest {
						err = validateRequest(requestPayload.Records[0].Cf.Config.EventType, requestPayload, respData)
						if err != nil {
							sendErrorResponse(w, "invalid response data", err.Error())
							return
						}
					}

					if respData.Status != nil {
						statusVal, err := strconv.Atoi(*respData.Status)
						if err != nil {
							sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *respData.Status), err.Error())
							return
						}

						writeRequestHeaders(w, respData)
						w.WriteHeader(statusVal)
						w.Write([]byte(respData.Body))
						return
					}
				}

				if event == ViewerResponse || event == OriginResponse {
					respData, err := parseResponseData(string(callbackContent))
					if err != nil {
						sendErrorResponse(w, "failed to parse response data", err.Error())
						return
					}

					finalResponse = mergeResponseResponseWithRequestPayload(finalResponse, requestPayload.Records[0].Cf.Response)

					// if respData.Headers != nil {
					// 	for key, val := range *respData.Headers {
					// 		headers := *requestPayload.Records[0].Cf.Request.Headers
					// 		headers[key] = val
					// 		requestPayload.Records[0].Cf.Request.Headers = &headers
					// 	}
					// }

					if event == ViewerResponse {
						err = validateResponse(requestPayload.Records[0].Cf.Config.EventType, requestPayload, respData)
						if err != nil {
							sendErrorResponse(w, "invalid response data", err.Error())
							return
						}

						if respData.Status != nil {
							statusVal, err := strconv.Atoi(*respData.Status)
							if err != nil {
								sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *respData.Status), err.Error())
								return
							}

							writeResponseHeaders(w, respData)
							w.WriteHeader(statusVal)
							w.Write([]byte(respData.Body))
							return
						}
					}
				}

				continue
			}

			statusVal, err := strconv.Atoi(*finalResponse.Status)
			if err != nil {
				sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *finalResponse.Status), err.Error())
				return
			}

			for key, val := range *finalResponse.Headers {
				w.Header().Add(key, val[0].Value)
			}
			w.WriteHeader(statusVal)
			w.Write(finalResponse.Body)
		})
	}
	return handlers
}

func generateRoutesV2(config *types.CloudfrontConfig, eventHandlers map[types.EventType]types.CloudfrontEvent) *chi.Mux {
	handlers := chi.NewRouter()

	sort.Slice(config.Behaviors, func(i, j int) bool {
		return config.Behaviors[i].Path > config.Behaviors[j].Path
	})

	for _, behaviorValue := range config.Behaviors {
		// make a copy since behaviorValue is a pointer in the slice
		behavior := behaviorValue

		origin, ok := config.OriginConfigs[behavior.Origin]
		if !ok {
			logrus.Fatalf("bad configuration: behavior uses undefined origin: %s", behavior.Origin)
		}

		handlers.HandleFunc(behavior.Path, func(w http.ResponseWriter, r *http.Request) {

			// listen for callback content
			var callbackContent []byte
			callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error

				callbackContent, err = ioutil.ReadAll(r.Body)
				if err != nil {
					sendErrorResponse(w, "failed to parse callback content", err.Error())
					return
				}
			}))
			defer callback.Close()

			var finalResponse *types.CfResponse
			var err error

			requestId := uuid.New()
			w.Header().Add("x-lambda-emulator-requestId", requestId.String())

			requestPayload := generateRequestBodyV2(requestId, types.ViewerRequest, r)
			responsePayload := &types.CfResponse{}
			recordPayload := &types.RequestPayload{
				Records: []types.Record{
					{
						Cf: types.CfRecord{
							Config:   types.CfType{},
							Request:  requestPayload,
							Response: responsePayload,
						},
					},
				},
			}

			for eventType, eventHandler := range eventHandlers {
				// this needs to run regardless before we run origin-response
				if eventType == types.OriginResponse {
					finalResponse, err = origins.Request(&origins.OriginRequestConfig{
						HTTPRequest: r,
						CfRequest:   *recordPayload,
						Origin:      origin,
					})
					if err != nil {
						sendErrorResponse(w, "failed to make origin request", err.Error())
						return
					}

					recordPayload.Records[0].Cf.Response = finalResponse
				}

				// if we don't have a behavior to act on, continue to the next event type
				handlerContext, ok := behavior.Events[eventType]
				if !ok {
					continue
				}

				recordPayload.Records[0].Cf.Config.EventType = eventType

				payload, err := recordPayload.EncodeJSON()
				if err != nil {
					logrus.WithError(err).Fatal("something went wrong marshaling the request")
					sendErrorResponse(w, "failed to generate status", err.Error())
					return
				}

				cwd, err := os.Getwd()
				if err != nil {
					sendErrorResponse(w, "failed to get cwd", err.Error())
					return
				}

				if len(os.Args) >= 2 {
					cwd = os.Args[1]
				}

				resp, err := lambda.Run(lambda.LambdaExecution{
					Callback:         callback,
					Payload:          payload,
					WorkingDirectory: cwd,
					Context:          handlerContext,
				})
				if err != nil {
					sendErrorResponse(w, fmt.Sprintf("failed to execute the lambda\n%s", resp), err.Error())
					return
				}

				// TODO: validate response headeres and data aren't immutable according to AWS docs
				// - [x] Viewer Request
				// - [x] Origin Request
				// - [ ] Origin Response
				// - [ ] Viewer Response

				err = eventHandler.Execute(types.CloudfrontEventConfig{
					CallbackResponse: callbackContent,
					CfRequest:        requestPayload,
					CfResponse:       responsePayload,
				})
				if err != nil {
					sendErrorResponse(w, "failed to execute handler actions", err.Error())
					return
				}

				if requestPayload.Status != nil {
					statusVal, err := strconv.Atoi(*requestPayload.Status)
					if err != nil {
						sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *requestPayload.Status), err.Error())
						return
					}

					writeRequestHeadersV2(w, *requestPayload)
					w.WriteHeader(statusVal)
					w.Write([]byte(requestPayload.Body))
					return
				}

				if eventType == types.ViewerResponse || eventType == types.OriginResponse {
					respData, err := parseResponseDataV2(string(callbackContent))
					if err != nil {
						sendErrorResponse(w, "failed to parse response data", err.Error())
						return
					}

					finalResponse = mergeResponseResponseWithRequestPayloadV2(finalResponse, recordPayload.Records[0].Cf.Response)

					// if respData.Headers != nil {
					// 	for key, val := range *respData.Headers {
					// 		headers := *recordPayload.Records[0].Cf.Request.Headers
					// 		headers[key] = val
					// 		recordPayload.Records[0].Cf.Request.Headers = &headers
					// 	}
					// }

					if eventType == types.ViewerResponse {
						err = validateResponseV2(recordPayload.Records[0].Cf.Config.EventType, *recordPayload, respData)
						if err != nil {
							sendErrorResponse(w, "invalid response data", err.Error())
							return
						}

						if respData.Status != nil {
							statusVal, err := strconv.Atoi(*respData.Status)
							if err != nil {
								sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *respData.Status), err.Error())
								return
							}

							writeResponseHeadersV2(w, respData)
							w.WriteHeader(statusVal)
							w.Write([]byte(respData.Body))
							return
						}
					}
				}

				continue
			}

			statusVal, err := strconv.Atoi(*finalResponse.Status)
			if err != nil {
				sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *finalResponse.Status), err.Error())
				return
			}

			for key, val := range *finalResponse.Headers {
				w.Header().Add(key, val[0].Value)
			}
			w.WriteHeader(statusVal)
			w.Write(finalResponse.Body)
		})
	}
	return handlers
}

func mergeResponseResponseWithRequestPayload(finalResponse *CfResponse, respData *CfResponse) *CfResponse {
	if respData.URI != finalResponse.URI {
		finalResponse.URI = respData.URI
	}

	if respData.Headers != finalResponse.Headers {
		finalResponse.Headers = respData.Headers
	}
	return finalResponse
}

func mergeRequestResponseWithRequestPayload(requestPayload RequestPayload, respData CfRequest) RequestPayload {
	request := requestPayload.Records[0].Cf.Request
	if respData.URI != request.URI {
		request.URI = respData.URI
	}

	if respData.Headers != request.Headers {
		request.Headers = respData.Headers
	}
	return requestPayload
}

func mergeResponseResponseWithRequestPayloadV2(finalResponse *types.CfResponse, respData *types.CfResponse) *types.CfResponse {
	if respData.URI != finalResponse.URI {
		finalResponse.URI = respData.URI
	}

	if respData.Headers != finalResponse.Headers {
		finalResponse.Headers = respData.Headers
	}
	return finalResponse
}

func mergeRequestResponseWithRequestPayloadV2(requestPayload types.RequestPayload, respData types.CfRequest) *types.RequestPayload {
	request := requestPayload.Records[0].Cf.Request
	if respData.URI != request.URI {
		request.URI = respData.URI
	}

	if respData.Headers != request.Headers {
		request.Headers = respData.Headers
	}
	return &requestPayload
}

func executeLambda(event []byte, context Event, callback *httptest.Server) ([]byte, error) {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to cwd")
	}

	if len(os.Args) >= 2 {
		cwd = os.Args[1]
	}

	handlerDefinition := strings.Split(context.Handler, ".")
	pathToHandler := filepath.Clean(fmt.Sprintf("./%s/%s.js", context.Path, handlerDefinition[0]))

	command := fmt.Sprintf(`require('./%s').%s(%s, 'f', (error, response) => { 
		if (error) {
			throw new Error(error)
		}

		const req = http.request("%s", {
			method: "POST",
		})
		req.write(JSON.stringify(response))
		req.end()
	})`, pathToHandler, handlerDefinition[1], string(event), callback.URL)

	cmd := exec.Command("node", "-e", command)

	cmd.Dir = cwd

	resp, err := cmd.CombinedOutput()
	if err != nil {
		return resp, errors.Wrap(err, "failed to execute the command")
	}

	responseData := strings.Split(string(resp), "\n")
	if len(responseData) > 1 {
		for _, line := range responseData[:len(responseData)-1] {
			fmt.Println(line)
		}
	}

	return resp, nil
}

func executeLambdaV2(event []byte, context types.Event, callback *httptest.Server) ([]byte, error) {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to cwd")
	}

	if len(os.Args) >= 2 {
		cwd = os.Args[1]
	}

	handlerDefinition := strings.Split(context.Handler, ".")
	pathToHandler := filepath.Clean(fmt.Sprintf("./%s/%s.js", context.Path, handlerDefinition[0]))

	command := fmt.Sprintf(`require('./%s').%s(%s, 'f', (error, response) => { 
		if (error) {
			throw new Error(error)
		}

		const req = http.request("%s", {
			method: "POST",
		})
		req.write(JSON.stringify(response))
		req.end()
	})`, pathToHandler, handlerDefinition[1], string(event), callback.URL)

	cmd := exec.Command("node", "-e", command)

	cmd.Dir = cwd

	resp, err := cmd.CombinedOutput()
	if err != nil {
		return resp, errors.Wrap(err, "failed to execute the command")
	}

	responseData := strings.Split(string(resp), "\n")
	if len(responseData) > 1 {
		for _, line := range responseData[:len(responseData)-1] {
			fmt.Println(line)
		}
	}

	return resp, nil
}

type CfServer struct {
	Server        *http.Server
	Handler       http.Handler
	Wg            *sync.WaitGroup
	PathToCerts   string
	EventHandlers map[types.EventType]types.CloudfrontEvent
}

func (cf *CfServer) Refresh(config *types.CloudfrontConfig) {
	ctx := context.TODO()
	if err := cf.Server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to shutdown server")
	}
	cf.Handler = generateRoutesV2(config, cf.EventHandlers)
	logrus.Info("waiting for server to shutdown")

	// make sure the
	cf.Wg.Wait()

	// decalre a new server
	cf.Server = &http.Server{
		Addr: cf.Server.Addr,
	}

	startServerV2(cf)
}

func (cf *Cloudfront) Refresh(config *CloudfrontConfig) {
	ctx := context.TODO()
	if err := cf.server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to shutdown server")
	}
	cf.handler = generateRoutes(config)
	logrus.Info("waiting for server to shutdown")

	// make sure the
	cf.wg.Wait()

	// decalre a new server
	cf.server = &http.Server{
		Addr: cf.server.Addr,
	}

	startServer(cf)
}

func startServer(cf *Cloudfront) {

	if strings.Split(cf.server.Addr, ":")[1] == "443" {
		cf.server.Handler = cf.handler
		go func(cf *Cloudfront) {
			cf.wg.Add(1)
			defer cf.wg.Done()
			if err := cf.server.ListenAndServeTLS(fmt.Sprintf("%s/cert.pem", cf.pathToCerts), fmt.Sprintf("%s/key.pem", cf.pathToCerts)); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("shutting down https server")
			}
		}(cf)
		logrus.Info("Server Started 🚀")
	} else {
		cf.server.Handler = cf.handler
		go func(cf *Cloudfront) {
			cf.wg.Add(1)
			defer cf.wg.Done()
			if err := cf.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("shutting down http server")
			}
		}(cf)
		logrus.Info("Server Started 🚀")
	}
}

func startServerV2(cf *CfServer) {
	if strings.Split(cf.Server.Addr, ":")[1] == "443" {
		cf.Server.Handler = cf.Handler
		go func(cf *CfServer) {
			cf.Wg.Add(1)
			defer cf.Wg.Done()
			if err := cf.Server.ListenAndServeTLS(fmt.Sprintf("%s/cert.pem", cf.PathToCerts), fmt.Sprintf("%s/key.pem", cf.PathToCerts)); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("shutting down https server")
			}
		}(cf)

		logrus.Info("Server Started 🚀")
	} else {
		cf.Server.Handler = cf.Handler
		go func(cf *CfServer) {
			cf.Wg.Add(1)
			defer cf.Wg.Done()
			if err := cf.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("shutting down http server")
			}
		}(cf)
		logrus.Info("Server Started 🚀")
	}
}

func checkHeaders(eventType EventType, reqHeaders CfHeaderArray, respHeaders CfHeaderArray) error {
	for respHeaderKey, header := range respHeaders {
		if strings.ToLower(header[0].Key) != strings.ToLower(respHeaderKey) {
			return fmt.Errorf("got %s saw key value %s", header[0].Key, respHeaderKey)
		}

		if err := checkReadOnlyHeader(AlwaysReadOnlyHeaders, header, reqHeaders); err != nil {
			return errors.Wrap(err, "read only headeres were modified")
		}

		if eventType == ViewerRequest {
			if err := checkReadOnlyHeader(ViewerRequestReadOnlyHeaders, header, reqHeaders); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}

		if eventType == OriginRequest {
			if err := checkReadOnlyHeader(OriginRequestReadOnlyHeaders, header, reqHeaders); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}
	}

	return nil
}

func checkHeadersV2(eventType types.EventType, reqHeaders types.CfHeaderArray, respHeaders types.CfHeaderArray) error {
	for respHeaderKey, header := range respHeaders {
		if strings.ToLower(header[0].Key) != strings.ToLower(respHeaderKey) {
			return fmt.Errorf("got %s saw key value %s", header[0].Key, respHeaderKey)
		}

		if err := checkReadOnlyHeaderV2(types.AlwaysReadOnlyHeaders, header, reqHeaders); err != nil {
			return errors.Wrap(err, "read only headeres were modified")
		}

		if eventType == types.ViewerRequest {
			if err := checkReadOnlyHeaderV2(types.ViewerRequestReadOnlyHeaders, header, reqHeaders); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}

		if eventType == types.OriginRequest {
			if err := checkReadOnlyHeaderV2(types.OriginRequestReadOnlyHeaders, header, reqHeaders); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}
	}

	return nil
}

func sendErrorResponse(w http.ResponseWriter, content, payload string) {
	w.WriteHeader(502)
	w.Header().Add("content-type", "text/html")
	fmt.Fprintf(w, `<html><body><h1>502 Error</h1><hr /><p><em>If you're seeing this it means something went wrong executing the logic in your lambda... More context can be found below:</em></p><hr /><pre>%s</pre><hr /><pre>%s</pre></body></html>`, content, payload)
}

func (c *Cloudfront) ParseResponse() {

}

func generateCertsForSSL(host string) string {

	tmpFolder := os.TempDir()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logrus.Fatalf("Failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		logrus.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"edwardofclt/cloudfront-emulator"},
		},
		DNSNames:  []string{host},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(3 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		logrus.Fatalf("Failed to create certificate: %v", err)
	}

	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemCert == nil {
		logrus.Fatal("Failed to encode certificate to PEM")
	}
	if err := os.WriteFile(fmt.Sprintf("%s/cert.pem", tmpFolder), pemCert, 0644); err != nil {
		logrus.Fatal(err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		logrus.Fatalf("Unable to marshal private key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if pemKey == nil {
		logrus.Fatal("Failed to encode key to PEM")
	}
	if err := os.WriteFile(fmt.Sprintf("%s/key.pem", tmpFolder), pemKey, 0600); err != nil {
		logrus.Fatal(err)
	}

	return tmpFolder
}
