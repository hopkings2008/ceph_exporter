//   Copyright 2016 DigitalOcean
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package collectors

import (
	"encoding/json"
	"fmt"

	"github.com/ceph/go-ceph/rados"
)

// Conn interface implements only necessary methods that are used
// in this repository of *rados.Conn. This keeps rest of the implementation
// clean and *rados.Conn doesn't need to show up everywhere (it being
// more of an implementation detail in reality). Also it makes mocking
// easier for unit-testing the collectors.
type Conn interface {
	ReadDefaultConfigFile() error
	Connect() error
	Shutdown()
	MonCommand([]byte) ([]byte, string, error)
	PGCommand([]byte, []byte) ([]byte, string, error)
}

// Verify that *rados.Conn implements Conn correctly.
var _ Conn = &rados.Conn{}

// NoopConn is the stub we use for mocking rados Conn. Unit testing
// each individual collectors becomes a lot easier after that.
type NoopConn struct {
	output string
	cmdOut map[string]string
}

// The stub we use for testing should also satisfy the interface properties.
var _ Conn = &NoopConn{}

// NewNoopConn returns an instance of *NoopConn. The string that we want output
// at the end of the command we issue to Ceph is fixed and should be specified
// in the only input parameter.
func NewNoopConn(output string) *NoopConn {
	return &NoopConn{
		output: output,
		cmdOut: make(map[string]string),
	}
}

// NewNoopConnWithCmdOut returns an instance of *NoopConn. The string that we
// want output at the end of the command we issue to Ceph can be various and
// should be specified by the map in the only input parameter.
func NewNoopConnWithCmdOut(cmdOut map[string]string) *NoopConn {
	return &NoopConn{cmdOut: cmdOut}
}

// ReadDefaultConfigFile does not need to return an error. It satisfies
// rados.Conn's function with the same prototype.
func (n *NoopConn) ReadDefaultConfigFile() error {
	return nil
}

// Connect does not need to return an error. It satisfies
// rados.Conn's function with the same prototype.
func (n *NoopConn) Connect() error {
	return nil
}

// Shutdown satisfies rados.Conn's function prototype.
func (n *NoopConn) Shutdown() {}

// MonCommand returns the provided output string to NoopConn as is, making
// it seem like it actually ran something and produced that string as a result.
func (n *NoopConn) MonCommand(args []byte) ([]byte, string, error) {
	// Unmarshal the input command and see if we need to intercept
	cmd := map[string]interface{}{}
	err := json.Unmarshal(args, &cmd)
	if err != nil {
		return []byte(n.output), "", err
	}

	// Intercept and mock the output
	switch prefix := cmd["prefix"]; prefix {
	case "pg dump":
		dc, ok := cmd["dumpcontents"].([]interface{})
		if !ok || len(dc) == 0 {
			break
		}

		switch dc[0] {
		case "pgs_brief":
			return []byte(n.cmdOut["ceph pg dump pgs_brief"]), "", nil
		}

	case "osd tree":
		st, ok := cmd["states"].([]interface{})
		if !ok || len(st) == 0 {
			break
		}

		switch st[0] {
		case "down":
			return []byte(n.cmdOut["ceph osd tree down"]), "", nil
		}

	case "osd df":
		return []byte(n.cmdOut["ceph osd df"]), "", nil

	case "osd perf":
		return []byte(n.cmdOut["ceph osd perf"]), "", nil

	case "osd dump":
		return []byte(n.cmdOut["ceph osd dump"]), "", nil
	}

	return []byte(n.output), "", nil
}

// PGCommand returns the provided output string to NoopConn as is, making
// it seem like it actually ran something and produced that string as a result.
func (n *NoopConn) PGCommand(pgid, args []byte) ([]byte, string, error) {
	// Unmarshal the input command and see if we need to intercept
	cmd := map[string]interface{}{}
	err := json.Unmarshal(args, &cmd)
	if err != nil {
		return []byte(n.output), "", err
	}

	// Intercept and mock the output
	switch prefix := cmd["prefix"]; prefix {
	case "query":
		return []byte(n.cmdOut[fmt.Sprintf("ceph tell %s query", string(pgid))]), "", nil
	}

	return []byte(n.output), "", nil
}
