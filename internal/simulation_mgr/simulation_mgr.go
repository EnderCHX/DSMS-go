package simulation_mgr

const (
	eventName = "simulation"
)

var gMap [][]int

type SimulationMgr struct {
	eventChan chan string
}
