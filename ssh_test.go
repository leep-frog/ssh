package ssh

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func TestSSHExecution(t *testing.T) {
	for _, test := range []struct {
		name string
		gsh  *GSH
		etc  *command.ExecuteTestCase
		want *GSH
	}{
		{
			name: "Sets environment variables if already exists",
			gsh: &GSH{
				AgentPID:   "123",
				AuthSocket: "some-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// checkProcess (also verify stdout and stderr isn't outputted).
					{
						Stdout: []string{"stdout output"},
						Stderr: []string{"stderr output"},
					},
				},
				WantRunContents: [][]string{
					{
						// TODO: These "sets" shouldn't be included by default
						"set -e",
						"set -o pipefail",
						`ps -p "123"`,
					},
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`export SSH_AGENT_PID="123"`,
						`export SSH_AUTH_SOCKET="some-file"`,
					},
				},
			},
		},
		{
			name: "Creates new ssh agent if agent died",
			gsh: &GSH{
				AgentPID:   "123",
				AuthSocket: "some-file",
			},
			want: &GSH{
				AgentPID:   "789",
				AuthSocket: "some-other-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// checkProcess
					{
						Err: fmt.Errorf("ps failed"),
					},
					// Create agent
					{
						Stdout: []string{
							"789",
							"some-other-file",
						},
						Stderr: []string{
							"blah blah",
						},
					},
				},
				WantRunContents: [][]string{
					// checkProcess
					{
						"set -e",
						"set -o pipefail",
						`ps -p "123"`,
					},
					// Create agent
					{
						"set -e",
						"set -o pipefail",
						createAgentContents,
					},
				},
				// TODO: Should this have a newline?
				WantStderr: "blah blah",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`export SSH_AGENT_PID="789"`,
						`export SSH_AUTH_SOCKET="some-other-file"`,
						`ssh-add`,
					},
				},
			},
		},
		{
			name: "Creates new ssh agent if AgentPID is empty",
			gsh: &GSH{
				AuthSocket: "some-file",
			},
			want: &GSH{
				AgentPID:   "789",
				AuthSocket: "some-other-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// Create agent
					{
						Stdout: []string{
							"789",
							"some-other-file",
						},
					},
				},
				WantRunContents: [][]string{
					// Create agent
					{
						"set -e",
						"set -o pipefail",
						createAgentContents,
					},
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`export SSH_AGENT_PID="789"`,
						`export SSH_AUTH_SOCKET="some-other-file"`,
						`ssh-add`,
					},
				},
			},
		},
		{
			name: "Creates new ssh agent if AuthSocket is empty",
			gsh: &GSH{
				AgentPID: "123",
			},
			want: &GSH{
				AgentPID:   "789",
				AuthSocket: "some-other-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// Create agent
					{
						Stdout: []string{
							"789",
							"some-other-file",
						},
					},
				},
				WantRunContents: [][]string{
					// Create agent
					{
						"set -e",
						"set -o pipefail",
						createAgentContents,
					},
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`export SSH_AGENT_PID="789"`,
						`export SSH_AUTH_SOCKET="some-other-file"`,
						`ssh-add`,
					},
				},
			},
		},
		{
			name: "Fails if error creating bash",
			gsh: &GSH{
				AgentPID:   "123",
				AuthSocket: "some-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// checkProcess
					{
						Err: fmt.Errorf("ps failed"),
					},
					// Create agent
					{
						Stdout: []string{
							"un",
							"deux",
							"trois",
						},
					},
				},
				WantRunContents: [][]string{
					// checkProcess
					{
						"set -e",
						"set -o pipefail",
						`ps -p "123"`,
					},
					// Create agent
					{
						"set -e",
						"set -o pipefail",
						createAgentContents,
					},
				},
				WantErr:    fmt.Errorf(`failed to create new ssh agent: validation for "" failed: [Length] length must be exactly 2`),
				WantStderr: "failed to create new ssh agent: validation for \"\" failed: [Length] length must be exactly 2\n",
			},
		},
		{
			name: "Fails if too few lines returned",
			gsh: &GSH{
				AgentPID:   "123",
				AuthSocket: "some-file",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{
					// checkProcess
					{
						Err: fmt.Errorf("ps failed"),
					},
					// Create agent
					{
						Err: fmt.Errorf("oopsie"),
					},
				},
				WantRunContents: [][]string{
					// checkProcess
					{
						"set -e",
						"set -o pipefail",
						`ps -p "123"`,
					},
					// Create agent
					{
						"set -e",
						"set -o pipefail",
						createAgentContents,
					},
				},
				WantErr:    fmt.Errorf("failed to create new ssh agent: failed to execute bash command: oopsie"),
				WantStderr: "failed to create new ssh agent: failed to execute bash command: oopsie\n",
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.gsh == nil {
				test.gsh = CLI()
			}
			test.etc.Node = test.gsh.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, test.want, test.gsh, cmpopts.IgnoreUnexported(GSH{}))
		})
	}
}

func TestMetadata(t *testing.T) {
	g := CLI()
	if g.Name() != "gsh" {
		t.Errorf("Name mismatch")
	}

	if g.Setup() != nil {
		t.Errorf("Setup mismatch")
	}
}
