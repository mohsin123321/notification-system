package pure

import "fmt"

func RoutingKey(ch Channel, p Priority) string {
	return fmt.Sprintf("notify.%s.%s", ch, p)
}
