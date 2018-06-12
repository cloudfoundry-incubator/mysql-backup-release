package galera_agent

type GaleraAgentCallerInterface interface {
	NodeWithHighestWsrepLocalIndex([]string) string
}

type GaleraAgentCaller struct {
}

func DefaultGaleraAgentCaller() GaleraAgentCallerInterface {
	return &GaleraAgentCaller{}
}

func (galeraAgentCaller *GaleraAgentCaller) NodeWithHighestWsrepLocalIndex([]string) string {
	return "HI!"
}
