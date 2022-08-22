package lambda

import (
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

type Package struct {
	Type string `json:"type"`
}

type LambdaExecution struct {
	Callback         *httptest.Server
	WorkingDirectory string
	Context          types.Event
	Payload          []byte
}

func Run(config LambdaExecution) ([]byte, error) {
	var err error

	handlerDefinition := strings.Split(config.Context.Handler, ".")
	pathToHandler := filepath.Clean(fmt.Sprintf("./%s/%s.js", config.Context.Path, handlerDefinition[0]))

	// packageFilePath := filepath.Join(config.WorkingDirectory, "package.json")
	// packageFile := &Package{}
	// packageFileContent, err := os.ReadFile(packageFilePath)
	// if err != nil && err != os.ErrInvalid {
	// 	return nil, err
	// }

	// if packageFileContent != nil {
	// 	err := json.Unmarshal(packageFileContent, packageFile)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if packageFile.Type == "module" {
	// 	return nil, fmt.Errorf("lambda uses modules")
	// }

	command := fmt.Sprintf(`require('./%s').%s(%s, 'f', async (error, response) => {
		if (error) {
			throw new Error(error)
		}

		const req = http.request("%s", {
			method: "POST",
		})
		req.write(JSON.stringify(response))
		req.end()	
	})`, pathToHandler, handlerDefinition[1], string(config.Payload), config.Callback.URL)

	cmd := exec.Command("node", "-e", command)

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
