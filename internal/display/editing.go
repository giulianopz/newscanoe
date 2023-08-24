package display

type buffer struct {
	chars  []rune
	offset int
}

func (b buffer) fitTo(width int) string {
	limit := b.offset + width
	if limit > len(b.chars) {
		limit = len(b.chars)
	}
	return string(b.chars[b.offset:limit])
}

func (b buffer) String() string {
	return string(b.chars)
}

func (b *buffer) insert(c byte, x int) bool {
	var inserted bool

	i := x + b.offset - 1

	if i == len(b.chars) {
		b.chars = append(b.chars, rune(c))
		inserted = true
	} else if i <= len(b.chars)-1 {
		b.chars = append(b.chars[:i+1], b.chars[i:]...)
		b.chars[i] = rune(c)
		inserted = true
	}
	return inserted
}

func (b *buffer) delete(x int) bool {
	var deleted bool

	i := x + b.offset - 1

	if i < 0 {
		return deleted
	} else if i == len(b.chars)-1 {
		b.chars[len(b.chars)-1] = NULL
		b.chars = b.chars[:len(b.chars)-1]
		deleted = true
	} else if i < len(b.chars)-1 {
		copy(b.chars[i:], b.chars[i+1:])
		b.chars[len(b.chars)-1] = NULL
		b.chars = b.chars[:len(b.chars)-1]
		deleted = true
	}
	return deleted
}

func (b *buffer) inBounds(x int) bool {
	i := x + b.offset - 1
	return i >= 0 && i <= len(b.chars)-1
}
