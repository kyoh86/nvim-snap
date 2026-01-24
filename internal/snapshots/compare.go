package snapshots

import "reflect"

func Equal(a, b Snapshot) bool {
	return reflect.DeepEqual(a, b)
}
