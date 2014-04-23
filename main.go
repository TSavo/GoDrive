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
	"os/exec"
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

func send_turbo(writer *bufio.Writer) {
	write_msg(writer, "turbo", "Pow!")
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
	data := make(map[string]interface{})
	data["name"] = name
	data["key"] = key
	data["trackName"] = "germany"
	data["carCount"] = 1
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

func DefineInstructions(throttle *float32, sw *int, turbo *int) (i *govirtual.InstructionSet) {
	i = govirtual.NewInstructionSet()
	govirtual.EmulationInstructions(i)
	i.Instruction("setThrottle", func(p *govirtual.Processor, args ...govirtual.Pointer) govirtual.Memory {
		defer func() {
			recover()
		}()
		*throttle = float32(govirtual.Cardinalize(args[0].Get())) / 1000.0
		if *throttle < 0 {
			*throttle = 0
		}
		if *throttle > 1 {
			*throttle = 1
		}
		return nil
	}, govirtual.Argument{"throttle", "int"})
	i.Instruction("switchLeft", func(p *govirtual.Processor, args ...govirtual.Pointer) govirtual.Memory {
		*sw = -1
		return nil
	})
	i.Instruction("switchRight", func(p *govirtual.Processor, args ...govirtual.Pointer) govirtual.Memory {
		*sw = 1
		return nil
	})
	i.Instruction("dontSwitch", func(p *govirtual.Processor, args ...govirtual.Pointer) govirtual.Memory {
		*sw = 0
		return nil
	})
	i.Instruction("turbo", func(p *govirtual.Processor, args ...govirtual.Pointer) govirtual.Memory {
		*turbo = 1
		return nil
	})

	return
}

type DrivingEvaluator struct {
	RaceSession *RaceSession
}

var driverIsland *goevolve.IslandEvolver = goevolve.NewIslandEvolver()

func (eval DrivingEvaluator) Evaluate(p *govirtual.Processor) int {
	timePenalty := eval.RaceSession.ElapsedTicks
	if timePenalty == 0 {
		timePenalty = 10000000000
	}
	if timePenalty > 1000000 {
		log.Printf("Reward for #%v: No Reward\n", p.Id)
	} else {
		log.Printf("Reward for #%v: %v\n", p.Id, timePenalty)
	}
	return timePenalty
}

type RaceSession struct {
	Heap           *govirtual.Memory
	DeadChannel    *govirtual.ChannelTerminationCondition
	Throttle       float32
	SwitchState    int
	Game           *GameInitMessage
	Velocity       float64
	Angle          float64
	LastPosition   float64
	StartTime      int
	ElapsedTicks   int
	NeedsToDie     bool
	Cost           int
	TurboOn        int
	TurboAvailable int
	SendTurbo      int
}

type DieAfterCondition struct {
	RaceSession *RaceSession
}

func (dieAfter *DieAfterCondition) ShouldTerminate(p *govirtual.Processor) bool {
	dieAfter.RaceSession.Cost++
	if goevolve.Now() > dieAfter.RaceSession.StartTime+int((10*60*time.Second)) || (dieAfter.RaceSession.Velocity < 3 && goevolve.Now()-int((5*time.Second)) > dieAfter.RaceSession.StartTime) {
		dieAfter.RaceSession.NeedsToDie = true
		dieAfter.RaceSession.StartTime = goevolve.Now()
		dieAfter.RaceSession.ElapsedTicks = 100000000
		fmt.Println("Dead velocity:", dieAfter.RaceSession.Velocity)
		return true
	} else {
		return false
	}
}

func NewRaceSession() *RaceSession {
	heap := make(govirtual.Memory, 30)
	deadChannel := govirtual.NewChannelTerminationCondition()
	race := RaceSession{&heap, deadChannel, 0.1, 0, nil, 0.0, 0.0, 0.0, goevolve.Now(), 0, false, 0, 0, 0, 0}
	return &race
}

func (session *RaceSession) NextDriver() {
	*session.DeadChannel <- true
}

func (session *RaceSession) StartSimulation() {
	is := DefineInstructions(&session.Throttle, &session.SwitchState, &session.SendTurbo)
	terminationCondition := govirtual.OrTerminate(session.DeadChannel, &DieAfterCondition{session})
	breeder := goevolve.Breeders(new(DriverProgramGenerator), goevolve.NewCopyBreeder(10), goevolve.NewRandomBreeder(25, 100, is), goevolve.NewMutationBreeder(25, 0.1, is), goevolve.NewCrossoverBreeder(25))
	selector := goevolve.AndSelect(goevolve.TopX(10), goevolve.Tournament(10))
	drivingEval := goevolve.Inverse(DrivingEvaluator{session})
	driverIsland.AddPopulation(session.Heap, 16, is, terminationCondition, breeder, drivingEval, selector)
}

type DriverProgramGenerator struct{}

func (d *DriverProgramGenerator) Breed(seeds []string) []string {
	dat, err := ioutil.ReadFile("bestProgram.vm")
	if err == nil {
		return []string{string(dat)}
	}
	return nil
}

func (session *RaceSession) Dispatch(writer *bufio.Writer, msgtype string, data interface{}, msg string) (err error) {
	switch msgtype {
	case "join":
		send_ping(writer)
	case "gameStart":
		send_ping(writer)
	case "crash":
		//session.Crash()
		session.ElapsedTicks = 10000000000
		session.NextDriver()
		err = errors.New("Crashed")
		send_ping(writer)
	case "spawn":
		send_ping(writer)
	case "gameEnd":
		var gameEnd GameEndMessage
		json.Unmarshal([]byte(msg), &gameEnd)
		err = errors.New("Game ended")
		session.ElapsedTicks = int(gameEnd.Data.Results[0].Result.Ticks) + int(gameEnd.Data.Results[0].Result.Millis)
		if session.ElapsedTicks == 0 {
			session.ElapsedTicks = 1000000000
		}
		session.NextDriver()
		return
	case "carPositions":
		if session.NeedsToDie {
			session.NeedsToDie = false
			err = errors.New("Needed to die!")
			return
		}
		var position CarPositionMessage
		json.Unmarshal([]byte(msg), &position)
		//angle := position.Data[0].Angle
		piece := session.Game.Data.Race.Track.Pieces[position.Data[0].PiecePosition.PieceIndex]
		nextPiece := session.Game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex+1)%len(session.Game.Data.Race.Track.Pieces)]
		pieceAfter := session.Game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex+2)%len(session.Game.Data.Race.Track.Pieces)]

		if position.Data[0].PiecePosition.InPieceDistance-session.LastPosition > 0 {
			session.Velocity = position.Data[0].PiecePosition.InPieceDistance - session.LastPosition
		}
		lastAngle := session.Angle
		session.Angle = position.Data[0].Angle
		angleDiff := 0.0
		if session.Angle < 0 && session.Angle < lastAngle {
			angleDiff = lastAngle - session.Angle
		} else if session.Angle > 0 && session.Angle > lastAngle {
			angleDiff = session.Angle - lastAngle
		}
		session.LastPosition = position.Data[0].PiecePosition.InPieceDistance
		(*session.Heap)[0].Set(&govirtual.Literal{int(session.Throttle * 1000)})
		(*session.Heap)[1].Set(&govirtual.Literal{int(session.Velocity * 1000)})
		(*session.Heap)[2].Set(&govirtual.Literal{int(angleDiff * 100)})
		(*session.Heap)[3].Set(&govirtual.Literal{int(position.Data[0].PiecePosition.InPieceDistance)})
		(*session.Heap)[4].Set(&govirtual.Literal{int(position.Data[0].PiecePosition.PieceIndex)})
		(*session.Heap)[5].Set(&govirtual.Literal{int(piece.Length)})
		(*session.Heap)[6].Set(&govirtual.Literal{int(piece.Angle)})
		(*session.Heap)[7].Set(&govirtual.Literal{int(piece.Radius)})
		(*session.Heap)[8].Set(&govirtual.Literal{int(nextPiece.Length)})
		(*session.Heap)[9].Set(&govirtual.Literal{int(nextPiece.Angle)})
		(*session.Heap)[10].Set(&govirtual.Literal{int(nextPiece.Radius)})
		(*session.Heap)[11].Set(&govirtual.Literal{int(pieceAfter.Length)})
		(*session.Heap)[12].Set(&govirtual.Literal{int(pieceAfter.Angle)})
		(*session.Heap)[13].Set(&govirtual.Literal{int(pieceAfter.Radius)})
		(*session.Heap)[14].Set(&govirtual.Literal{int(session.Angle)})
		(*session.Heap)[15].Set(&govirtual.Literal{int(session.TurboOn)})
		(*session.Heap)[16].Set(&govirtual.Literal{int(session.TurboAvailable)})
		(*session.Heap)[17].Set(&govirtual.Literal{int(session.SendTurbo)})
		//fmt.Print(msg)
		ms, _ := time.ParseDuration("10ms")
		cost := session.Cost
		for cost+20 < session.Cost {
			time.Sleep(ms)
		}
		//fmt.Println((*session.Heap)[1], (*session.Heap)[2], piece.Length, nextPiece.Length, session.Throttle)
		if session.SwitchState == -1 {
			switch_left(writer)
		} else if session.SwitchState == 1 {
			switch_right(writer)
		}
		session.SwitchState = 0
		if session.SendTurbo == 1 && session.TurboAvailable == 1 {
			session.SendTurbo = 0
			session.TurboAvailable = 0
			send_turbo(writer)
		}
		send_throttle(writer, session.Throttle)
	case "error":
		log.Printf(fmt.Sprintf("Got error: %v", data))
		send_ping(writer)
	case "gameInit":
		game := new(GameInitMessage)
		json.Unmarshal([]byte(msg), &game)
		session.Game = game
		session.Throttle = 0
		session.Velocity = 0
		session.Angle = 0
		session.ElapsedTicks = 0
		session.Cost = 0
		session.TurboOn = 0
		session.TurboAvailable = 0
		session.SendTurbo = 0
		session.StartTime = goevolve.Now()
		session.Heap.Zero()
		send_ping(writer)
	case "lapFinished":
		send_ping(writer)
	case "yourCar":
		send_ping(writer)
	case "turboAvailable":
		send_ping(writer)
		session.TurboAvailable = 1
	case "turboStart":
		send_ping(writer)
		session.TurboOn = 1
	case "turboEnd":
		send_ping(writer)
		session.TurboOn = 0
	default:
		log.Printf("Got msg type: %s: %v", msgtype, data)
		send_ping(writer)
	}
	return
}

func (session *RaceSession) parse_and_dispatch_input(writer *bufio.Writer, input interface{}, message string) (err error) {
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

func (session *RaceSession) bot_loop(conn net.Conn, name string, key string) (err error) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	defer recover()
	send_join(writer, name, key)
	for {
		input, msg, err := read_msg(reader)
		if err != nil {
			//log_and_exit(err)
			return nil
		}
		err = session.parse_and_dispatch_input(writer, input, msg)
		if err != nil {
			//log_and_exit(err)
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
	fmt.Println(err)
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
		go func(id int) {
			session := NewRaceSession()
			session.StartSimulation()
			c := make(chan bool)
			for {
				go func() {
					defer func() {
						recover()
						c <- true
					}()
					session.StartTime = goevolve.Now()
					conn, err := connect(host, port)
					session.StartTime = goevolve.Now()
	
					if err != nil {
						log_and_exit(err)
						return
					}

					defer func() {
						defer func(){
							recover()
						}()
						conn.Close()
					}()

					err = session.bot_loop(conn, name+strconv.Itoa(id), key)
				}()
				<-c
			}

		}(x)
	}
	go func() {
		for {
			out, _ := exec.Command("git", "pull").Output()
			log.Printf("Git: %v", string(out))
			time.Sleep(60 * time.Second)
		}
	}()
	<-make(chan bool, 0)
}
