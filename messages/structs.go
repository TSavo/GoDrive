package messages

type CarPositionMessage struct {
	MsgType string
	Data    []CarPosition
}

type CarPosition struct {
	Id            CarId
	Angle         float64
	PiecePosition PiecePosition
}

type PiecePosition struct {
	PieceIndex      int
	InPieceDistance float64
	Lane            LanePosition
	Lap             int
}

type LanePosition struct {
	StartLaneIndex int
	EndLaneIndex   int
}

//{"msgType":"gameEnd","data":{"results":[{"car":{"name":"evowork","color":"red"},"result":{"laps":3,"
//ticks":3848,"millis":64134}}],"bestLaps":[{"car":{"name":"evowork","color":"red"},"result":{"lap":0,"ticks":1011,"millis
//":16850}}]},"gameId":"558fcbb7-c5ac-418c-92b4-08ab4b8e2749"}

type GameEndMessage struct {
	MsgType string
	Data    GameEnd
}

type GameEnd struct {
	Results  []LapSummary
	BestLaps []LapSummary
}

type LapSummary struct {
	Car    CarId
	Result LapResult
}

type LapResult struct {
	Laps, Lap, Ticks, Millis int
}

type GameInitMessage struct {
	MsgType string
	Data    GameInit
}

type GameInit struct {
	Race Race
}

type Race struct {
	Track Track
	Cars  []Car
}

type Track struct {
	Id, Name      string
	Pieces        []Piece
	Lanes         []Lane
	StartingPoint StartingPoint
}

type Piece struct {
	Length, Radius, Angle float64
	Switch                bool
}

type Lane struct {
	DistanceFromCenter float64
	Index              int
}

type StartingPoint struct {
	Position Point
	Angle    float64
}

type Point struct {
	X, Y float64
}

type Car struct {
	Id         CarId
	Dimensions Dimensions
}

type CarId struct {
	Name, Color string
}

type Dimensions struct {
	Length, Width, GuideFlagPosition float64
}
