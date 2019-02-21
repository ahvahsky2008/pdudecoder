# PduDecoder

# Ported from js version: jsPduDecoder of Benjamin Erhart, berhart@tladesignz.com

# Usage
`go get github.com/vinhjaxt/pdudecoder`

```go
import (
  "hex"
  "fmt"
  "log"

  "github.com/vinhjaxt/pdudecoder"
)

func dumpMsg(msg *pdudecoder.Message) {
	fmt.Println("Type:", msg.Type)
	fmt.Println("SMSC:", msg.SMSC)
	fmt.Println("Address:", msg.Address)
	fmt.Println("ServiceCenterTime:", msg.ServiceCenterTime)
	fmt.Println("Text:", msg.Text)
	fmt.Println("ValidityPeriod:", msg.ValidityPeriod)
	fmt.Println("PartNumber:", msg.PartNumber)
	fmt.Println("TotalParts:", msg.TotalParts)
	fmt.Printf("\r\n\r\n")
}


func main (){
  bs, err := hex.DecodeString(`07914889200009F50406D0B11B0C00009120221041658249D17A1EB44687C768D0185D0F83C861F719B4CE83E67510B9EE3E83CEEF34683A7381ACF53488FD769F41EB74B90DA2CBC3207638ED0261D36ED038DC06BDDD21`)
  if err != nil {
    log.Println(err)
    return
  }
  msg, err := pdudecoder.Decode(bs)
  if err != nil {
    log.Println(err)
    return
  }
  dumpMsg(msg)
}


```