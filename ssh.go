package ssh

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command"
)

const (
	agentPidEnv   = "SSH_AGENT_PID"
	authSocketEnv = "SSH_AUTH_SOCK"
	killContents  = "ps | grep ssh-agent | awk '{ print $1 }' | xargs kill > /dev/null 2>&1"
)

var (
	createAgentContents = fmt.Sprintf("eval `ssh-agent` > /dev/null && echo $%s && echo $%s", agentPidEnv, authSocketEnv)
)

func CLI() *GSH {
	return &GSH{}
}

type GSH struct {
	AgentPID   string
	AuthSocket string
	changed    bool
}

func (g *GSH) Name() string {
	return "gsh"
}

func (g *GSH) Setup() []string { return nil }
func (g *GSH) Changed() bool   { return g.changed }

func (g *GSH) Node() *command.Node {
	return command.AsNode(&command.BranchNode{
		Branches: map[string]*command.Node{
			"kill k": command.SerialNodes(
				command.SimpleExecutableNode(killContents),
			),
		},
		Default: command.SerialNodes(
			command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
				// If a process already exists, then just point to that process.
				if g.checkProcess() {
					return []string{
						fmt.Sprintf("export %s=%q", agentPidEnv, g.AgentPID),
						fmt.Sprintf("export %s=%q", authSocketEnv, g.AuthSocket),
					}, nil
				}

				fmt.Println("Creating ssh agent")

				// Create new ssh agent
				bc := &command.BashCommand[[]string]{
					Contents:      []string{createAgentContents},
					ForwardStdout: true,
					Validators: []*command.ValidatorOption[[]string]{
						command.Length[string, []string](2),
					},
				}
				vars, err := bc.Run(o, d)
				fmt.Println("Response", vars, " |||| ", err)
				if err != nil {
					return nil, o.Annotatef(err, "failed to create new ssh agent")
				}
				g.AgentPID, g.AuthSocket = vars[0], vars[1]
				g.changed = true

				// Set environment variables and run ssh-add
				return []string{
					fmt.Sprintf("export %s=%q", agentPidEnv, g.AgentPID),
					fmt.Sprintf("export %s=%q", authSocketEnv, g.AuthSocket),
					"ssh-add",
				}, nil
			}),
		),
	})
}

func (g *GSH) checkProcess() bool {
	if strings.TrimSpace(g.AgentPID) == "" || strings.TrimSpace(g.AuthSocket) == "" {
		return false
	}
	bc := &command.BashCommand[[]string]{
		Contents: []string{
			fmt.Sprintf("ps -p %q", g.AgentPID),
		},
	}
	fmt.Println("Running ps command")
	_, err := bc.Run(nil, &command.Data{})
	return err == nil
}
