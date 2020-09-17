package redis

type Writer struct {
	ReadWrite
	CurrentScore uint
}

func (w *Writer) Write(p []byte) (int, error) {
	// Get data at the current score
	lines, err := w.ReadWrite.get(w.CurrentScore, w.CurrentScore)
	if err != nil {
		return 0, err
	}
	var currentLine string
	if len(lines) == 1 {
		currentLine = lines[0]
	}

	var n int

	for _, bch := range p {
		charact := string(bch)
		currentLine = currentLine + charact
		n++
		if charact == "\n" {
			if err := w.ReadWrite.add(w.CurrentScore, currentLine); err != nil {
				return 0, err
			}
			w.CurrentScore++
			currentLine = ""
		}
	}

	// Save into redis current non-finished line
	if len(currentLine) > 0 {
		if err := w.ReadWrite.add(w.CurrentScore, currentLine); err != nil {
			return 0, err
		}
	}

	return n, nil
}
