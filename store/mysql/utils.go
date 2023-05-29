package mysql

func Chunk[T any](ss []T, chunkLength int) [][]T {
	if chunkLength <= 0 {
		panic("chunkLength should be greater than 0")
	}

	result := make([][]T, 0)
	l := len(ss)
	if l == 0 {
		return result
	}

	var step = l / chunkLength
	if step == 0 {
		result = append(result, ss)
		return result
	}
	var remain = l % chunkLength
	for i := 0; i < step; i++ {
		result = append(result, ss[i*chunkLength:(i+1)*chunkLength])
	}
	if remain != 0 {
		result = append(result, ss[step*chunkLength:l])
	}

	return result
}

func IntersectStringSlice(a []string, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := make(map[string]bool)
	var intersect []string
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			intersect = append(intersect, v)
		}
	}
	return intersect
}
