package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/TSavo/GoDrive/messages"
	"github.com/TSavo/GoEvolve"
	"github.com/TSavo/GoVirtual"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func connect(host string, port int) (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	return
}

func read_msg(reader *bufio.Reader) (msg interface{}, line string, err error) {
	line, err = reader.ReadString('\n')
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(line), &msg)
	if err != nil {
		return
	}
	return
}

func write_msg(writer *bufio.Writer, msgtype string, data interface{}) (err error) {
	m := make(map[string]interface{})
	m["msgType"] = msgtype
	m["data"] = data
	var payload []byte
	payload, err = json.Marshal(m)
	_, err = writer.Write([]byte(payload))
	if err != nil {
		return
	}
	_, err = writer.WriteString("\n")
	if err != nil {
		return
	}
	writer.Flush()
	return
}

func send_join(writer *bufio.Writer, name string, key string) (err error) {
	data := make(map[string]string)
	data["name"] = name
	data["key"] = key
	err = write_msg(writer, "join", data)
	return
}

func send_ping(writer *bufio.Writer) (err error) {
	err = write_msg(writer, "ping", make(map[string]string))
	return
}

func send_throttle(writer *bufio.Writer, throttle float32) (err error) {
	err = write_msg(writer, "throttle", throttle)
	return
}

func switch_left(writer *bufio.Writer) (err error) {
	err = write_msg(writer, "switchLane", "Left")
	return
}
func switch_right(writer *bufio.Writer) (err error) {
	err = write_msg(writer, "switchLane", "Right")
	return
}

func DefineInstructions(throttle *float32, sw *int) (i *govirtual.InstructionSet) {
	i = govirtual.NewInstructionSet()
	govirtual.EmulationInstructions(i)
	i.Instruction("setThrottle", func(p *govirtual.Processor, m *govirtual.Memory) {
		*throttle = float32(p.Registers.Get(m.Get(0))) / 1000.0
		if *throttle < 0 {
			*throttle = 0
		}
		if *throttle > 1 {
			*throttle = 1
		}
	})
	i.Instruction("switchLeft", func(p *govirtual.Processor, m *govirtual.Memory) {
		*sw = -1
	})
	i.Instruction("switchRight", func(p *govirtual.Processor, m *govirtual.Memory) {
		*sw = 1
	})
	i.Instruction("dontSwitch", func(p *govirtual.Processor, m *govirtual.Memory) {
		*sw = 0
	})

	return
}

type DrivingEvaluator struct {
	RaceSession *RaceSession
}

var driverIsland *goevolve.IslandEvolver = goevolve.NewIslandEvolver()

func (eval *DrivingEvaluator) Evaluate(p *govirtual.Processor) int64 {
	timePenalty := 100000 - (time.Now().UnixNano() - eval.RaceSession.StartTime) / (int64(time.Second) / 1000)
	fmt.Println(timePenalty)
	return timePenalty
}

func GenerateProgram() string {
	return ""
}

type RaceSession struct {
	Heap             *govirtual.Memory
	DeadChannel      *govirtual.ChannelTerminationCondition
	//DieAfter         *govirtual.TimeTerminationCondition
	NeedToSpawn      bool
	Throttle         float32
	SwitchState      int
	Game             *GameInitMessage
	Velocity         float64
	LastPosition     float64
	//DistanceTraveled float64
	//Crashed          bool
	//LapFinished      int
	StartTime				 int64
}

func NewRaceSession() *RaceSession {
	heap := make(govirtual.Memory, 30)
	deadChannel := govirtual.NewChannelTerminationCondition()
	race := RaceSession{&heap, deadChannel, false, 0.1, 0, nil, 0.0, 0.0, time.Now().UnixNano()}
	return &race
}

func (session *RaceSession) NextDriver() {
	*session.DeadChannel <- true
}

func (session *RaceSession) Spawn() {
	session.NeedToSpawn = false
	//session.DieAfter.Reset()
}

func (session *RaceSession) StartSimulation() {
	is := DefineInstructions(&session.Throttle, &session.SwitchState)
	//terminationCondition := govirtual.OrTerminate(session.DeadChannel, session.DieAfter)
	breeder := *goevolve.Breeders(new(DriverProgramGenerator), goevolve.NewCopyBreeder(15), goevolve.NewRandomBreeder(25, 100, is), goevolve.NewMutationBreeder(25, 0.1, is), goevolve.NewCrossoverBreeder(25))
	selector := goevolve.AndSelect(goevolve.TopX(10), goevolve.Tournament(10))
	drivingEval := &DrivingEvaluator{session}
	driverIsland.AddPopulation(session.Heap, nil, 8, is, session.DeadChannel, breeder, drivingEval, selector)
}

type DriverProgramGenerator struct{}

func (d *DriverProgramGenerator) Breed(seeds []string) []string {
	m := make([]string, 1)
	dat, err := ioutil.ReadFile("bestProgram.vm")
	if err == nil {
		m[0] = string(dat)
		return m
	}
	for i := 0; i < len(m); i++ {
		m[i] = `
set 0,100
setThrottle 0
jump 7
noop
noop
noop
noop
noop
noop
jump 7		
`
	}
	return m
}

func (session *RaceSession) Dispatch(writer *bufio.Writer, msgtype string, data interface{}, msg string) (err error) {
	switch msgtype {
	case "join":
		log.Printf("Joined")
		send_ping(writer)
	case "gameStart":
		log.Printf("%v", msg)
		send_ping(writer)
	case "crash":
		//session.Crash()
		send_ping(writer)
	case "spawn":
		session.Spawn()
		send_ping(writer)
	case "gameEnd":
		log.Printf("Game ended")
		err = errors.New("Game ended")
		session.NextDriver()
		return
	case "carPositions":
		var position CarPositionMessage
		json.Unmarshal([]byte(msg), &position)
		//angle := position.Data[0].Angle
		piece := session.Game.Data.Race.Track.Pieces[position.Data[0].PiecePosition.PieceIndex]
		nextPiece := session.Game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex+1)%len(session.Game.Data.Race.Track.Pieces)]
		pieceAfter := session.Game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex+2)%len(session.Game.Data.Race.Track.Pieces)]
		session.Velocity = position.Data[0].PiecePosition.InPieceDistance - session.LastPosition
		session.LastPosition = position.Data[0].PiecePosition.InPieceDistance
		(*session.Heap)[0] = int(session.Throttle * 1000)
		(*session.Heap)[1] = int(session.Velocity * 1000)
		(*session.Heap)[2] = int(position.Data[0].Angle)
		(*session.Heap)[3] = int(position.Data[0].PiecePosition.InPieceDistance)
		(*session.Heap)[4] = int(position.Data[0].PiecePosition.PieceIndex)
		(*session.Heap)[5] = int(piece.Length)
		(*session.Heap)[6] = int(piece.Angle)
		(*session.Heap)[7] = int(piece.Radius)
		(*session.Heap)[8] = int(nextPiece.Length)
		(*session.Heap)[9] = int(nextPiece.Angle)
		(*session.Heap)[10] = int(nextPiece.Radius)
		(*session.Heap)[11] = int(pieceAfter.Length)
		(*session.Heap)[12] = int(pieceAfter.Angle)
		(*session.Heap)[13] = int(pieceAfter.Radius)
		if session.SwitchState == -1 {
			switch_left(writer)
		} else if session.SwitchState == 1 {
			switch_right(writer)
		}
		session.SwitchState = 0
		send_throttle(writer, session.Throttle)
	case "error":
		log.Printf(fmt.Sprintf("Got error: %v", data))
		send_ping(writer)
	case "gameInit":
		game := new(GameInitMessage)
		json.Unmarshal([]byte(msg), &game)
		session.Game = game
		send_ping(writer)
		session.StartTime = time.Now().UnixNano()
	case "lapFinished":
		send_ping(writer)
	default:
		log.Printf("Got msg type: %s: %v", msgtype, data)
		send_ping(writer)
	}
	return
}

func parse_and_dispatch_input(session *RaceSession, writer *bufio.Writer, input interface{}, message string) (err error) {
	switch t := input.(type) {
	default:
		err = errors.New(fmt.Sprintf("Invalid message type: %T", t))
		return
	case map[string]interface{}:
		var msg map[string]interface{}
		var ok bool
		msg, ok = input.(map[string]interface{})
		if !ok {
			err = errors.New(fmt.Sprintf("Invalid message type: %v", msg))
			return
		}
		switch msg["data"].(type) {
		default:
			err = session.Dispatch(writer, msg["msgType"].(string), nil, message)
			if err != nil {
				return
			}
		case interface{}:
			err = session.Dispatch(writer, msg["msgType"].(string), msg["data"].(interface{}), message)
			if err != nil {
				return
			}
		}
	}
	return
}

func bot_loop(session *RaceSession, conn net.Conn, name string, key string) (err error) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	defer recover()
	send_join(writer, name, key)
	for {
		input, msg, err := read_msg(reader)
		if err != nil {
			log_and_exit(err)
			return nil
		}
		err = parse_and_dispatch_input(session, writer, input, msg)
		if err != nil {
			log_and_exit(err)
			return nil
		}
	}
}

func parse_args() (host string, port int, name string, key string, err error) {
	args := os.Args[1:]
	if len(args) != 4 {
		return "", 0, "", "", errors.New("Usage: ./run host port botname botkey")
	}
	host = args[0]
	port, err = strconv.Atoi(args[1])
	if err != nil {
		return "", 0, "", "", errors.New(fmt.Sprintf("Could not parse port value to integer: %v\n", args[1]))
	}
	name = args[2]
	key = args[3]

	return
}

func log_and_exit(err error) {
	//log.Fatal(err)
	fmt.Println("log and exit", err)
	//os.Exit(1)
}

func main() {

	host, port, name, key, err := parse_args()

	if err != nil {
		log_and_exit(err)
		return
	}

	fmt.Println("Connecting with parameters:")
	fmt.Printf("host=%v, port=%v, bot name=%v, key=%v\n", host, port, name, key)

	for x := 0; x < 10; x++ {
		go func() {
			session := NewRaceSession()
			session.StartSimulation()

			for {
				conn, err := connect(host, port)

				if err != nil {
					log_and_exit(err)
				}

				defer conn.Close()

				err = bot_loop(session, conn, name, key)
				fmt.Println("Again")
				session.NeedToSpawn = false
			}

		}()
	}
	<-make(chan bool, 0)
}
