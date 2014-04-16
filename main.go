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

func DefineInstructions(lookAhead *int, throttle *float32) (i *govirtual.InstructionSet) {
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
		fmt.Println(*throttle)
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

var lastAngle = 0
var throttle = float32(0.1)
var game GameInitMessage
var velocity = 0.0
var lastPosition = 0.0
var distanceTraveled = 0.0
var crashed = false

var accTimingDone = false

type DrivingEvaluator struct{}

var driverIsland *goevolve.IslandEvolver = goevolve.NewIslandEvolver(3)

func (eval *DrivingEvaluator) Evaluate(p *govirtual.Processor) int64 {
	x := distanceTraveled
	distanceTraveled = 0.0
	throttle = 0.1
	timeTerminationCondition.Reset()
	if crashed {
		crashed = false
		fmt.Println("Crashed! No reward.")
		return 0
	}
	fmt.Println(x)
	return int64(x)
}

func GenerateProgram() string {
	return ""
}

var populationInfluxChan goevolve.InfluxBreeder = make(goevolve.InfluxBreeder, 100)
var PopulationReportChan chan *goevolve.PopulationReport = make(chan *goevolve.PopulationReport, 100)

var lookAhead int
var heap = make(govirtual.Memory, 20)
var floatHeap = make(govirtual.FloatMemory, 8)
var deadChannel = govirtual.NewChannelTerminationCondition()
var timeTerminationCondition = govirtual.NewTimeTerminationCondition(10 * time.Second)
var restart func()
var needToSpawn = false

func StartSimulation() {
	is := DefineInstructions(&lookAhead, &throttle)
	terminationCondition := govirtual.OrTerminate(deadChannel, timeTerminationCondition)
	breeder := *goevolve.Breeders(new(DriverProgramGenerator), goevolve.NewCopyBreeder(15), goevolve.NewRandomBreeder(25, 50, is), goevolve.NewMutationBreeder(25, 0.1, is), goevolve.NewCrossoverBreeder(25))
	divingEval := new(DrivingEvaluator)
	selector := goevolve.AndSelect(goevolve.TopX(10), goevolve.Tournament(10))
	driverIsland.AddPopulation(&heap, &floatHeap, 8, is, terminationCondition, breeder, divingEval, selector)
}

type DriverProgramGenerator struct {}

func (d *DriverProgramGenerator) Breed(seeds []string) []string {
	m := make([]string, 1)
	for i := 0; i < len(m); i++ {
		m[i] = `
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
speedUp
jump 13
noop
noop
noop
noop
noop
noop
jump 13		
`
	}
	return m
}

func dispatch_msg(writer *bufio.Writer, msgtype string, data interface{}, msg string) (err error) {
	switch msgtype {
	case "join":
		log.Printf("Joined")
		send_ping(writer)
	case "gameStart":
		log.Printf("%v", msg)
		send_ping(writer)
	case "crash":
		log.Printf("%v", data)
		crashed = true
		(*deadChannel) <- true
		fmt.Println("Crash message recieved")
		needToSpawn = true
		send_ping(writer)
	case "spawn":
		needToSpawn = false
		fmt.Println("Spawn message recieved")
		timeTerminationCondition.Reset()
		send_ping(writer)
	case "gameEnd":
		log.Printf("Game ended")
		err = errors.New("Game ended")
		return
		// Exit is not strictly necessary?
		//os.Exit(0)
//		restart()
	case "carPositions":
		if(needToSpawn) {
			timeTerminationCondition.Reset()
		}
		var position CarPositionMessage
		json.Unmarshal([]byte(msg), &position)
		//angle := position.Data[0].Angle
		piece := game.Data.Race.Track.Pieces[position.Data[0].PiecePosition.PieceIndex]
		nextPiece := game.Data.Race.Track.Pieces[(position.Data[0].PiecePosition.PieceIndex + 1) % len(game.Data.Race.Track.Pieces)]
		lastVelocity := velocity
		velocity = position.Data[0].PiecePosition.InPieceDistance - lastPosition
		//acceleration := velocity - lastVelocity
		lastPosition = position.Data[0].PiecePosition.InPieceDistance
		if velocity > 0 {
			distanceTraveled += velocity
			//fmt.Printf("%v %v\n", velocity, acceleration) //, angle, position.Data[0].PiecePosition.PieceIndex, piece)
		} else {
			velocity = lastVelocity
		}
		heap[0] = int(velocity * 100)
		heap[1] = int(position.Data[0].Angle)
		heap[2] = int(position.Data[0].PiecePosition.InPieceDistance)
		heap[3] = int(piece.Length)
		heap[4] = int(piece.Angle)
		heap[5] = int(piece.Radius)
		heap[6] = int(nextPiece.Length)
		heap[7] = int(nextPiece.Angle)
		heap[8] = int(nextPiece.Radius)

		send_throttle(writer, throttle)
	case "error":
		log.Printf(fmt.Sprintf("Got error: %v", data))
		send_ping(writer)
	case "gameInit":
		json.Unmarshal([]byte(msg), &game)
		send_ping(writer)
	default:
		log.Printf("Got msg type: %s: %v", msgtype, data)
		send_ping(writer)
	}
	return
}

func parse_and_dispatch_input(writer *bufio.Writer, input interface{}, message string) (err error) {
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
			err = dispatch_msg(writer, msg["msgType"].(string), nil, message)
			if err != nil {
				return
			}
		case interface{}:
			err = dispatch_msg(writer, msg["msgType"].(string), msg["data"].(interface{}), message)
			if err != nil {
				return
			}
		}
	}
	return
}

func bot_loop(conn net.Conn, name string, key string) (err error) {
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
		err = parse_and_dispatch_input(writer, input, msg)
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

	StartSimulation()

	for {
		conn, err := connect(host, port)

		if err != nil {
			log_and_exit(err)
		}

		defer conn.Close()

		err = bot_loop(conn, name, key)
		fmt.Println("Again")
	}
}
