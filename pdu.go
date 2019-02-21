package pdudecoder

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/vinhjaxt/pdudecoder/decoder"
)

// MessageType byte represent msg type
type MessageType byte

// MessageDeliver is value for MessageType
const MessageDeliver = 0

// MessageSubmit is value for MessageType
const MessageSubmit = 1

// Message struct of data
type Message struct {
	Type              uint8
	SMSC              string
	Address           string
	ServiceCenterTime string
	ValidityPeriod    string
	Text              string
	PartNumber        uint8
	TotalParts        uint8
}

func blocks(n, block int) int {
	if n%block == 0 {
		return n / block
	}
	return n/block + 1
}

// DecodeNumber decode phone number
func DecodeNumber(octets []byte, ntype byte) (string, error) {
	if ntype == 0x50 {
		return decoder.Decode7Bit(octets)
	}
	return decoder.DecodeSemiAddress(octets), nil
}

// DCSDefault is default type of Data Coding Scheme
const DCSDefault = uint8(0x1)

// DCS8Bit is a type of Data Coding Scheme
const DCS8Bit = uint8(0x2)

// DCSUC2 is a type of Data Coding Scheme
const DCSUC2 = uint8(0x3)

// DataCodingScheme return type of Data Coding Scheme
func DataCodingScheme(o byte) uint8 {
	var alphabet = DCSDefault
	var codingGroup = o & 0xF0
	if codingGroup >= 0 && codingGroup <= 0x30 {
		var alphabetFlag = o & 0xC
		if alphabetFlag == 0 {
		} else if alphabetFlag == 4 {
			alphabet = DCS8Bit
		} else if alphabetFlag == 8 {
			alphabet = DCSUC2
		}
	} else if codingGroup == 0xF0 {
		// noinspection JSBitwiseOperatorUsage
		if o&4 != 0 {
			alphabet = DCS8Bit
		}
	}
	return alphabet
}

// ValidityPeriodRelative return time.Now plus Relative ValidityPeriod
func ValidityPeriodRelative(b byte) string {
	vp := int64(b)
	if vp < 144 {
		return strconv.FormatInt((vp+1)*5, 10) + " minutes"
	} else if vp > 143 && vp < 168 {
		return strconv.FormatInt((vp-143)*30/60+12, 10) + " hours"
	} else if vp > 167 && vp < 197 {
		return strconv.FormatInt(vp-166, 10) + " days"
	} else if vp > 186 {
		return strconv.FormatInt(vp-192, 10) + " weeks"
	}
	return ""
}

// ServiceCentreTimeStamp return time
func ServiceCentreTimeStamp(o []byte) string {
	var octets = make([]int64, len(o))
	for i := 0; i < 7; i++ {
		octets[i] = int64(decoder.Decode(decoder.Swap(o[i])))
	}
	var ts = ""
	if octets[0] < 70 {
		ts += "20"
	} else {
		ts += "19"
	}

	ts += fmt.Sprintf("%02d", octets[0]) + "-" + fmt.Sprintf("%02d", octets[1]) + "-" + fmt.Sprintf("%02d", octets[2]) + " " + fmt.Sprintf("%02d", octets[3]) + ":" + fmt.Sprintf("%02d", octets[4]) + ":" + fmt.Sprintf("%02d", octets[5]) + " GMT "

	var tz = octets[6]

	// noinspection JSBitwiseOperatorUsage
	if tz&0x80 == 0 {
		ts += "+"
	} else {
		tz = tz & 0x7F
		ts += "-"
	}

	return ts + strconv.FormatInt(tz/4, 10)
}

// UserDataLength return User Data Length
func UserDataLength(o byte, alphabet uint8) int {
	var length = 0
	if alphabet == DCSDefault {
		length = int(math.Ceil(float64(o) * 70 / 80))
	} else {
		length = int(o)
	}
	return length
}

func pad7Bit(v byte, padding uint8) byte {
	return byte((v >> padding) & 0xFF)
}

// SIE struct of User Data Header
type SIE struct {
	IEI  int
	IEDL int
	IED  []byte
}

// UserDataHeader process User Data Header
func UserDataHeader(octets []byte) []*SIE {
	var IE = &SIE{
		IEI:  -1,
		IEDL: -1,
	}
	var IEs []*SIE

	for i := 0; i < len(octets); i++ {
		o := octets[i]
		if IE.IEI == -1 {
			IE.IEI = int(o) // Information Element Identifier
		} else if IE.IEDL == -1 {
			IE.IEDL = int(o) // Information Element Data Length
		} else {
			IE.IED = append(IE.IED, o)
			if len(IE.IED) >= IE.IEDL {
				IEs = append(IEs, IE)
				IE = &SIE{
					IEI:  -1,
					IEDL: -1,
				}
			}
		}
	}

	return IEs
}

// UserData decode User Data
func UserData(octets []byte, alphabet, padding uint8) (string, error) {
	if alphabet == DCSUC2 {
		return decoder.DecodeUcs2(octets, false)
	}
	if padding == 0 {
		return decoder.Decode7Bit(octets)
	}

	str, err := decoder.Decode7Bit(octets[1:])
	str2, err := decoder.Decode7Bit([]byte{pad7Bit(octets[0], padding)})
	if err != nil {
		// return
	}
	str = str2 + str
	return str, err
}

// Decode message by octets bytes
func Decode(octets []byte) (msg *Message, err error) {
	defer func() {
		err2 := recover()
		if err2 != nil {
			err = errors.New(fmt.Sprint(err2))
		}
	}()
	msg = &Message{}
	pos := 0

	// smsc
	if smscLength := int(octets[0]); smscLength != 0 {
		smsc, err := DecodeNumber(octets[2:smscLength+1], octets[1]&0x70)
		if err != nil {
			return nil, err
		}
		msg.SMSC = smsc
		pos = smscLength
	}
	pos++

	// Sender/Receiver part
	var tpDCS uint8
	var pduTypeByte = octets[pos]
	if pduTypeByte&0x1 == 0 {
		msg.Type = MessageDeliver
		// DELIVER
		pos++
		numberLength := int(octets[pos])
		pos++
		if numberLength != 0 {
			sliceNumberToA := octets[pos]
			numberByteLength := 1 + int(math.Ceil(float64(numberLength)/2))
			// sliceNumber := octets[pos+1 : pos+blocks(numberLength, 2)+1]
			sliceNumber := octets[pos+1 : pos+numberByteLength]
			addr, err := DecodeNumber(sliceNumber, sliceNumberToA&0x70)
			if err != nil {
				return nil, err
			}
			msg.Address = addr
			pos += numberByteLength
		}
		pos++
		tpDCS = DataCodingScheme(octets[pos])
		pos++
		msg.ServiceCenterTime = ServiceCentreTimeStamp(octets[pos : pos+7])
		pos += 6
	} else {
		// SUBMIT
		msg.Type = MessageSubmit
		pos += 2
		numberLength := int(octets[pos])
		pos++
		if numberLength != 0 {
			sliceNumber := octets[pos+1 : pos+1+int(math.Ceil(float64(numberLength)/2))]
			// sliceNumber := octets[pos+1 : pos+blocks(numberLength, 2)+2]
			sliceNumberToA := octets[pos]
			number, err := DecodeNumber(sliceNumber, sliceNumberToA&0x70)
			if err != nil {
				return nil, err
			}
			msg.Address = number
			pos += 1 + int(math.Ceil(float64(numberLength)/2))
		}
		pos++
		tpDCS = DataCodingScheme(octets[pos])
		var vpType = pduTypeByte & 0x18
		if vpType != 0 {
			pos++
			if vpType == 0x10 {
				msg.ValidityPeriod = ValidityPeriodRelative(octets[pos])
			} else if vpType == 0x18 {
				msg.ValidityPeriod = ServiceCentreTimeStamp(octets[pos : pos+7])
				pos += 6
			}
		}
	}
	pos++

	var tpUDL = UserDataLength(octets[pos], tpDCS)
	var msgLength = int(octets[pos])
	if tpDCS == DCSUC2 {
		msgLength /= 2
	}

	var tpUDHL int
	if pduTypeByte&0x40 != 0 {
		pos++
		tpUDHL = int(octets[pos])
		pos++
		tpUDH := UserDataHeader(octets[pos : pos+tpUDHL])
		for _, IE := range tpUDH {
			if IE.IEI == 0 {
				msg.PartNumber = IE.IED[2]
				msg.TotalParts = IE.IED[1]
			}
		}
		pos += tpUDHL - 1
	}

	var paddingUDHL uint8
	lengthUDHL := 0
	if tpUDHL != 0 {
		lengthUDHL = tpUDHL + 1
		var udhBitLength = lengthUDHL * 8
		var nextSeptetStart = int(math.Ceil(float64(udhBitLength)/7) * 7)

		paddingUDHL = uint8(nextSeptetStart - udhBitLength)
		msgLength -= lengthUDHL + 1
	}
	pos++

	var expectedMsgEnd = pos + tpUDL - lengthUDHL
	var sliceMessage = octets[pos:expectedMsgEnd]
	userData, err := UserData(sliceMessage, tpDCS, paddingUDHL)
	if err != nil {
		return nil, err
	}

	if expectedMsgEnd < len(octets) {
		//PDU longer than expected
		var sliceMessageAll = octets[pos:len(octets)]
		plusData, _ := UserData(sliceMessageAll, tpDCS, 0)
		userData += plusData
	} else if expectedMsgEnd > len(octets) {
		// PDU shorter than expected
	}

	if len(userData) > msgLength {
		msg.Text = userData[0:msgLength]
	} else {
		msg.Text = userData
	}

	return msg, nil
}
