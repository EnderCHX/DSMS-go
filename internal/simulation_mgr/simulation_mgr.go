package simulation_mgr

const (
	eventName = "simulation"
)

type SimulationMgr struct {
	eventChan chan string
}
