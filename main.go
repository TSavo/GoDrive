package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/TSavo/GoDrive/messages"
	"github.com/TSavo/GoEvolve"
	"github.com/TSavo/GoVirtual"
	"log"
	"math"
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

func DefineInstructions(throttle *float32) (i *govirtual.InstructionSet) {
	i = govirtual.NewInstructionSet()
	i.Instruction("noop", func(p *govirtual.Processor, m *govirtual.Memory) {
	})
	i.Movement("jump", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Jump(m.Get(0))
	})
	i.Movement("jumpIfZero", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) == 0 {
			p.Jump(m.Get(1))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("jumpIfNotZero", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) != 0 {
			p.Jump(m.Get(1))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("jumpIfEquals", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) == p.Registers.Get((*m).Get(1)) {
			p.Jump(m.Get(2))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("jumpIfNotEquals", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) != p.Registers.Get((*m).Get(1)) {
			p.Jump(m.Get(2))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("jumpIfGreaterThan", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) > p.Registers.Get((*m).Get(1)) {
			p.Jump(m.Get(2))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("jumpIfLessThan", func(p *govirtual.Processor, m *govirtual.Memory) {
		if p.Registers.Get((*m).Get(0)) < p.Registers.Get((*m).Get(1)) {
			p.Jump(m.Get(2))
		} else {
			p.InstructionPointer++
		}
	})
	i.Movement("call", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Call((*m).Get(0))
	})
	i.Movement("return", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Return()
	})
	i.Instruction("set", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Set((*m).Get(0), (*m).Get(1))
	})
	i.Instruction("store", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Heap.Set(m.Get(0), p.Registers.Get(m.Get(1)))
	})
	i.Instruction("load", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Set(m.Get(0), p.Heap.Get(m.Get(1)))
	})
	i.Instruction("swap", func(p *govirtual.Processor, m *govirtual.Memory) {
		x := p.Registers.Get((*m).Get(0))
		p.Registers.Set((*m).Get(0), (*m).Get(1))
		p.Registers.Set((*m).Get(1), x)
	})
	i.Instruction("push", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Stack.Push(p.Registers.Get((*m).Get(0)))
	})
	i.Instruction("pop", func(p *govirtual.Processor, m *govirtual.Memory) {
		if x, err := p.Stack.Pop(); !err {
			p.Registers.Set((*m).Get(0), x)
		}
	})
	i.Instruction("increment", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Increment((*m).Get(0))
	})
	i.Instruction("decrement", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Decrement((*m).Get(0))
	})
	i.Instruction("add", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Set((*m).Get(0), p.Registers.Get((*m).Get(0))+p.Registers.Get((*m).Get(1)))
	})
	i.Instruction("subtract", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.Registers.Set((*m).Get(0), p.Registers.Get((*m).Get(0))-p.Registers.Get((*m).Get(1)))
	})
	i.Instruction("speedUp", func(p *govirtual.Processor, m *govirtual.Memory) {
		*throttle += 0.01
		if *throttle > 1 {
			*throttle = 1
		}
	})
	i.Instruction("slowDown", func(p *govirtual.Processor, m *govirtual.Memory) {
		*throttle -= 0.01
		if *throttle < 0 {
			*throttle = 0
		}
	})
	i.Instruction("setThrottle", func(p *govirtual.Processor, m *govirtual.Memory) {
		*throttle = float32(p.Registers.Get(m.Get(0))) / 1000.0
		if *throttle < 0 {
			*throttle = 0
		}
		if *throttle > 1 {
			*throttle = 1
		}
	})

	//	AddMathInstructions(i)

	return
}

func AddMathInstructions(is *govirtual.InstructionSet) {
	is.Instruction("floatAdd", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), p.FloatHeap.Get(m.Get(0))+p.FloatHeap.Get(m.Get(1)))
	})
	is.Instruction("floatSubtract", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), p.FloatHeap.Get(m.Get(0))-p.FloatHeap.Get(m.Get(1)))
	})
	is.Instruction("floatMultiply", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), p.FloatHeap.Get(m.Get(0))*p.FloatHeap.Get(m.Get(1)))
	})
	is.Instruction("floatDivide", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), p.FloatHeap.Get(m.Get(0))/p.FloatHeap.Get(m.Get(1)))
	})
	is.Instruction("floatSet", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(0), float64(m.Get(1))*0.0000001)
	})
	is.Instruction("floatCopy", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), p.FloatHeap.Get(m.Get(0)))
	})
	is.Instruction("floatAbs", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Abs(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatAcos", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Acos(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatAcosh", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Acosh(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatAsin", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Asin(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatAsinh", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Asinh(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatCbrt", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Cbrt(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatCeil", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Ceil(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatCos", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Cos(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatDim", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Dim(p.FloatHeap.Get(m.Get(0)), math.Abs(p.FloatHeap.Get(m.Get(1)))))
	})
	is.Instruction("floatErf", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Erf(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatExp", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Exp(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatExp2", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Exp2(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatExpm1", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Abs(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatFloor", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Floor(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatGamma", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Gamma(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatHypot", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Hypot(p.FloatHeap.Get(m.Get(0)), math.Abs(p.FloatHeap.Get(m.Get(1)))))
	})
	is.Instruction("floatJ0", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.J0(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatJ1", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.J1(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatLog", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Log(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatLog10", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Log10(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatLog1p", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Log1p(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatLog2", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Log2(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatLogb", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Logb(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatMax", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Max(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatMin", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Min(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatMod", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Mod(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatNextAfter", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Nextafter(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatPow", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Pow(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatRemainder", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(2), math.Remainder(p.FloatHeap.Get(m.Get(0)), p.FloatHeap.Get(m.Get(1))))
	})
	is.Instruction("floatSin", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Sin(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatSinh", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Sinh(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatSqrt", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Sqrt(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatTan", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Tan(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatTanh", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Tanh(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatTrunc", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Trunc(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatY0", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Y0(p.FloatHeap.Get(m.Get(0))))
	})
	is.Instruction("floatY1", func(p *govirtual.Processor, m *govirtual.Memory) {
		p.FloatHeap.Set(m.Get(1), math.Y1(p.FloatHeap.Get(m.Get(0))))
	})
}

var accTimingDone = false

type DrivingEvaluator struct {
	RaceSession *RaceSession
}

var driverIsland *goevolve.IslandEvolver = goevolve.NewIslandEvolver(3)

func (eval *DrivingEvaluator) Evaluate(p *govirtual.Processor) int64 {
	x := eval.RaceSession.DistanceTraveled
	eval.RaceSession.DistanceTraveled = 0.0
	if eval.RaceSession.Crashed {
		fmt.Println("Crashed!")
		eval.RaceSession.Crashed = false
		fmt.Println(int64(x / 100))
		return int64(x / 100)
	}
	fmt.Println(int64(x))
	return int64(x)
}

func GenerateProgram() string {
	return ""
}

type RaceSession struct {
	Heap             *govirtual.Memory
	DeadChannel      *govirtual.ChannelTerminationCondition
	DieAfter         *govirtual.TimeTerminationCondition
	NeedToSpawn      bool
	Throttle         float32
	Game             *GameInitMessage
	Velocity         float64
	LastPosition     float64
	DistanceTraveled float64
	Crashed          bool
}

func NewRaceSession() *RaceSession {
	heap := make(govirtual.Memory, 20)
	deadChannel := govirtual.NewChannelTerminationCondition()
	timeTerminationCondition := govirtual.NewTimeTerminationCondition(10 * time.Second)
	race := RaceSession{&heap, deadChannel, timeTerminationCondition, false, 0.1, nil, 0.0, 0.0, 0.0, false}
	return &race
}

func (session *RaceSession) Crash() {
	session.Crashed = true
	session.NeedToSpawn = true
	*session.DeadChannel <- true
}

func (session *RaceSession) Spawn() {
	session.NeedToSpawn = false
	session.DieAfter.Reset()
}

func (session *RaceSession) StartSimulation() {
	is := DefineInstructions(&session.Throttle)
	terminationCondition := govirtual.OrTerminate(session.DeadChannel, session.DieAfter)
	breeder := *goevolve.Breeders(new(DriverProgramGenerator), goevolve.NewCopyBreeder(15), goevolve.NewRandomBreeder(25, 50, is), goevolve.NewMutationBreeder(25, 0.1, is), goevolve.NewCrossoverBreeder(25))
	selector := goevolve.AndSelect(goevolve.TopX(10), goevolve.Tournament(10))
	drivingEval := &DrivingEvaluator{session}
	driverIsland.AddPopulation(session.Heap, nil, 8, is, terminationCondition, breeder, drivingEval, selector)
}

type DriverProgramGenerator struct{}

func (d *DriverProgramGenerator) Breed(seeds []string) []string {
	m := make([]string, 1)
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
		session.Crash()
		send_ping(writer)
	case "spawn":
		session.Spawn()
		send_ping(writer)
	case "gameEnd":
		log.Printf("Game ended")
		err = errors.New("Game ended")
		return
	case "carPositions":
		if session.NeedToSpawn {
			session.DieAfter.Reset()
		}
		var position CarPositionMessage
		json.Unmarshal([]byte(msg), &position)
		//angle := position.Data[0].Angle
		piece := session.Game.Data.Race.Track.Pieces[position.Data[0].PiecePosition.PieceIndex]
		nextPiece := session.Game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex+1)%len(session.Game.Data.Race.Track.Pieces)]
		lastVelocity := session.Velocity
		session.Velocity = position.Data[0].PiecePosition.InPieceDistance - session.LastPosition
		session.LastPosition = position.Data[0].PiecePosition.InPieceDistance
		if session.Velocity > 0 {
			session.DistanceTraveled += session.Velocity
			//fmt.Printf("%v %v\n", velocity, acceleration) //, angle, position.Data[0].PiecePosition.PieceIndex, piece)
		} else {
			session.Velocity = lastVelocity
		}
		(*session.Heap)[0] = int(session.Velocity * 100)
		(*session.Heap)[1] = int(position.Data[0].Angle)
		(*session.Heap)[2] = int(position.Data[0].PiecePosition.InPieceDistance)
		(*session.Heap)[3] = int(piece.Length)
		(*session.Heap)[4] = int(piece.Angle)
		(*session.Heap)[5] = int(piece.Radius)
		(*session.Heap)[6] = int(nextPiece.Length)
		(*session.Heap)[7] = int(nextPiece.Angle)
		(*session.Heap)[8] = int(nextPiece.Radius)

		send_throttle(writer, session.Throttle)
	case "error":
		log.Printf(fmt.Sprintf("Got error: %v", data))
		send_ping(writer)
	case "gameInit":
		game := new(GameInitMessage)
		json.Unmarshal([]byte(msg), &game)
		session.Game = game
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
	<- make(chan bool, 0)
}
