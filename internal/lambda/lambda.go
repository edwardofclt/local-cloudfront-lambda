package lambda

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Package struct {
	Type string `json:"type"`
}

type LambdaExecution struct {
	Callback         *httptest.Server
	WorkingDirectory string
	Context          types.Event
	Payload          []byte
	Waitgroup        *sync.WaitGroup
}

const defaultLambdaCommand = `require('./{{.Path}}').{{.Handler}}({{.Payload}}, 'f', async (error, response) => {
	if (error) {
		throw new Error(error)
	}

	const req = http.request("{{.CallbackURL}}", {
		method: "POST",
	})
	req.write(JSON.stringify(response))
	req.end()	
})

const req = http.request("{{.CallbackURL}}", {
	method: "POST",
})
req.write(JSON.stringify({{.Payload}}))
req.end()`

const moduleLambdaCommand = `let module;
import('./{{.Path}}').then(m => m.{{.Handler}}({{.Payload}}, 'f', async (error, response) => {
	if (error) {
		throw new Error(error)
	}

	const req = http.request("{{.CallbackURL}}", {
		method: "POST",
	})
	req.write(JSON.stringify(response))
	req.end()	
})).then(() => {
	const req = http.request("{{.CallbackURL}}", {
		method: "POST",
	})
	req.write(JSON.stringify({{.Payload}}))
	req.end()	
});`

type LambdaTemplateValues struct {
	Path        string
	Handler     string
	Payload     string
	CallbackURL string
}

func Run(config LambdaExecution) ([]byte, error) {
	var err error

	cmd := new(exec.Cmd)

	handlerDefinition := strings.Split(config.Context.Handler, ".")

	packageFilePath := filepath.Join(config.WorkingDirectory, "package.json")
	packageFile := &Package{}
	packageFileContent, err := os.ReadFile(packageFilePath)
	if err == nil {
		err := json.Unmarshal(packageFileContent, packageFile)
		if err != nil {
			return nil, err
		}
	}

	templateValues := &LambdaTemplateValues{
		Path:        filepath.Clean(fmt.Sprintf("./%s/%s.js", config.Context.Path, handlerDefinition[0])),
		Handler:     handlerDefinition[1],
		Payload:     string(config.Payload),
		CallbackURL: config.Callback.URL,
	}

	command := &bytes.Buffer{}
	tmpl, err := template.New("command").Parse(defaultLambdaCommand)
	if err != nil {
		logrus.Fatal("failed to parse command template: %w", err)
	}
	tmpl.Execute(command, templateValues)

	if packageFile.Type == "module" {
		command.Reset()
		logrus.Info("Running as a module")
		tmpl, err := template.New("command").Parse(moduleLambdaCommand)
		if err != nil {
			logrus.Fatal("failed to parse module template: %w", err)
		}
		tmpl.Execute(command, templateValues)
	}

	cmd = exec.Command("node", "-e", command.String())
	cmd.Dir = config.WorkingDirectory
	resp, err := cmd.CombinedOutput()

	// output the logs from the lambda before throwing the error
	responseData := strings.Split(string(resp), "\n")
	if len(responseData) > 1 {
		for _, line := range responseData[:len(responseData)-1] {
			fmt.Println(line)
		}
	}

	if err != nil {
		return resp, errors.Wrap(err, "failed to execute the command")
	}

	return resp, nil
}
