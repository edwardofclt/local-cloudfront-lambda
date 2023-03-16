package cloudfront

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/davecgh/go-spew/spew"
	"github.com/edwardofclt/cloudfront-emulator/internal/lambda"
	originrequest "github.com/edwardofclt/cloudfront-emulator/internal/origin-request"
	originresponse "github.com/edwardofclt/cloudfront-emulator/internal/origin-response"
	"github.com/edwardofclt/cloudfront-emulator/internal/origins"
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	viewerrequest "github.com/edwardofclt/cloudfront-emulator/internal/viewer-request"
	viewerresponse "github.com/edwardofclt/cloudfront-emulator/internal/viewer-response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Event struct {
	Name    types.EventType
	Handler types.CloudfrontEvent
}

func New(config *types.CloudfrontConfig) *CfServer {
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

	eventHandlers := []Event{
		{
			Name:    types.ViewerRequest,
			Handler: viewerrequest.New(),
		},
		{
			Name:    types.OriginRequest,
			Handler: originrequest.New(),
		},
		{
			Name:    types.OriginResponse,
			Handler: originresponse.New(),
		},
		{
			Name:    types.ViewerResponse,
			Handler: viewerresponse.New(),
		},
	}

	cf := &CfServer{
		Server: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", addr, port),
			Handler: generateRoutes(config, eventHandlers),
		},
		Wg:            &sync.WaitGroup{},
		EventHandlers: eventHandlers,
	}

	if port == 443 {
		cf.PathToCerts = generateCertsForSSL(addr)
	}

	return cf
}

func (cf *CfServer) Start() {
	startServer(cf)
}

func generateRoutes(config *types.CloudfrontConfig, eventHandlers []Event) *chi.Mux {
	handlers := chi.NewRouter()

	sort.Slice(config.Behaviors, func(i, j int) bool {
		return config.Behaviors[i].Path > config.Behaviors[j].Path
	})

	for _, behaviorValue := range config.Behaviors {
		// make a copy since behaviorValue is a pointer in the slice
		behavior := behaviorValue

		handlers.HandleFunc(behavior.Path, func(w http.ResponseWriter, r *http.Request) {
			requestId := uuid.New()
			w.Header().Add("x-lambda-emulator-requestId", requestId.String())

			origin, ok := config.OriginConfigs[behavior.Origin]
			if !ok {
				err := fmt.Errorf("bad configuration: behavior uses undefined origin: %s requestId: %s", behavior.Origin, requestId)
				logrus.Error(err)
				w.Write([]byte(err.Error()))
			}

			var finalResponse *types.CfResponse
			var err error

			requestPayload := generateRequestBody(requestId, types.ViewerRequest, r)
			responsePayload := &types.CfResponse{}
			recordPayload := &types.RequestPayload{
				Records: []types.Record{
					{
						Cf: types.CfRecord{
							Config: types.CfType{
								DistributionId:   "E1234567890",
								RequestId:        requestId,
								DistributionName: "E1234567890",
							},
							Request:  requestPayload,
							Response: responsePayload,
						},
					},
				},
			}

			callbackContent := &types.CallbackResponse{}
			for _, eventHandler := range eventHandlers {
				wg := &sync.WaitGroup{}
				// In order to make the callback function work more like what (I think) the callback does
				// within AWS, we're going to make it actually callback to a server endpoint with POST data.
				// This simplifies the way we ingest the content and makes it easier to keep logs and
				// actual response data separate.
				wg.Add(1)

				alreadyCalled := false
				callback := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					data, err := io.ReadAll(r.Body)
					if err != nil {
						sendErrorResponse(w, "failed to parse callback content", err.Error())
						return
					}

					err = json.Unmarshal(data, callbackContent)
					if err != nil {
						sendErrorResponse(w, "failed to unmarshal callback content", err.Error())
						return
					}
					if !alreadyCalled {
						alreadyCalled = true
						wg.Done()
					}
				}))
				defer callback.Close()

				// We do this check because it's the origin is request immediately before OriginResponse
				if eventHandler.Name == types.OriginResponse {
					finalResponse, err = origins.Request(&origins.OriginRequestConfig{
						HTTPRequest: r,
						CfRequest:   *recordPayload,
						Origin:      origin,
					})
					if err != nil {
						sendErrorResponse(w, "failed to make origin request", err.Error())
						return
					}
				}

				// If the configuration isn't configured for this event type, go on to the next event type
				handlerContext, ok := behavior.Events[eventHandler.Name]
				if !ok {
					continue
				}
				recordPayload.Records[0].Cf.Config.EventType = eventHandler.Name

				payload, err := recordPayload.EncodeJSON()
				if err != nil {
					logrus.WithError(err).Fatal("something went wrong marshaling the request")
					sendErrorResponse(w, "failed to generate status", err.Error())
					return
				}

				resp, err := lambda.Run(lambda.LambdaExecution{
					Callback:         callback,
					Payload:          payload,
					WorkingDirectory: config.WorkingDirectory,
					Context:          handlerContext,
					Waitgroup:        wg,
				})
				if err != nil {
					sendErrorResponse(w, fmt.Sprintf("failed to execute the lambda\n%s", resp), err.Error())
					return
				}

				wg.Wait()

				config := types.CloudfrontEventInput{
					CallbackResponse: *callbackContent,
					CfRequest:        requestPayload,
					CfResponse:       responsePayload,
					FinalResponse:    finalResponse,
				}

				err = eventHandler.Handler.Execute(config)
				if err != nil {
					sendErrorResponse(w, "failed to execute handler actions", err.Error())
					return
				}

				if callbackContent.Status != nil {
					statusVal, err := strconv.Atoi(*callbackContent.Status)
					if err != nil {
						sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *callbackContent.Status), err.Error())
						return
					}

					writeRequestHeaders(w, *callbackContent.Headers)
					w.WriteHeader(statusVal)
					if callbackContent.Body != nil {
						w.Write([]byte(*callbackContent.Body))
					}
					return
				}
			}

			// Sanity checks that the headers are there
			types.MergeHeaders(finalResponse.Headers, callbackContent.Headers)

			statusVal, err := strconv.Atoi(*finalResponse.Status)
			if err != nil {
				sendErrorResponse(w, fmt.Sprintf("invalid status code: %s", *finalResponse.Status), err.Error())
				return
			}

			for key, val := range *finalResponse.Headers {
				w.Header().Add(key, val[0].Value)
			}
			w.WriteHeader(statusVal)
			if finalResponse.Body != nil {
				w.Write([]byte(*finalResponse.Body))
			}
		})
	}
	return handlers
}

func mergeResponseResponseWithRequestPayload(finalResponse *types.CfResponse, respData *types.CfResponse) *types.CfResponse {
	if finalResponse != nil {
	}
	if respData.URI != finalResponse.URI {
		finalResponse.URI = respData.URI
	}

	if respData.Status != nil {
		finalResponse.Status = respData.Status
	}

	if respData.Headers != nil {
		if respData.Headers != finalResponse.Headers {
			finalResponse.Headers = respData.Headers
		}
	}
	return finalResponse
}

func mergeRequestResponseWithRequestPayload(requestPayload types.RequestPayload, respData types.CfRequest) *types.RequestPayload {
	request := requestPayload.Records[0].Cf.Request
	if respData.URI != request.URI {
		request.URI = respData.URI
	}

	if respData.Headers != request.Headers {
		request.Headers = respData.Headers
	}
	return &requestPayload
}

type CfServer struct {
	Server        *http.Server
	Handler       http.Handler
	Wg            *sync.WaitGroup
	PathToCerts   string
	EventHandlers []Event
}

func (cf *CfServer) Refresh(config *types.CloudfrontConfig) {
	ctx := context.TODO()
	if err := cf.Server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to shutdown server")
	}
	logrus.Info("waiting for server to shutdown")

	// make sure the
	cf.Wg.Wait()

	// decalre a new server
	cf.Server = &http.Server{
		Addr:    cf.Server.Addr,
		Handler: generateRoutes(config, cf.EventHandlers),
	}

	startServer(cf)
}

func startServer(cf *CfServer) {
	if strings.Split(cf.Server.Addr, ":")[1] == "443" {
		go func(cf *CfServer) {
			cf.Wg.Add(1)
			defer cf.Wg.Done()
			if err := cf.Server.ListenAndServeTLS(fmt.Sprintf("%s/cert.pem", cf.PathToCerts), fmt.Sprintf("%s/key.pem", cf.PathToCerts)); err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("shutting down https server")
			}
		}(cf)

		logrus.Info("Server Started 🚀")
	} else {
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

func sendErrorResponse(w http.ResponseWriter, content, payload string) {
	w.WriteHeader(502)
	w.Header().Add("content-type", "text/html")
	fmt.Fprintf(w, `<html><body><h1>502 Error</h1><hr /><p><em>If you're seeing this it means something went wrong executing the logic in your lambda... More context can be found below:</em></p><hr /><pre>%s</pre><hr /><pre>%s</pre></body></html>`, content, payload)
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
