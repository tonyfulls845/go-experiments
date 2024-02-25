package frame

type OpCode byte

const (
	ContinuationFrame    byte = 0
	TextFrame            byte = 1
	BinaryFrame          byte = 2
	ConnectionCloseFrame byte = 8
	PingFrame            byte = 9
	PongFrame            byte = 10
)

type Frame struct {
	IsFin   bool
	OpCode  byte
	Length  uint64
	Payload []byte
}

func New(IsFin bool, OpCode byte, Length uint64, Payload []byte) *Frame {

	return &Frame{
		IsFin,
		OpCode,
		Length,
		Payload,
	}
}
