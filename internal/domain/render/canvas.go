package render

import (
	"strings"
)

// Canvas — неизменяемая растровая матрица символов
type Canvas struct {
	width  int
	height int
	buffer []rune // row-major: buffer[y*width + x]
}

// NewCanvas создаёт новый канвас заданного размера, заполненный паддингом
func NewCanvas(width, height int, padding rune) Canvas {
	size := width * height
	buf := make([]rune, size)
	for i := range buf {
		buf[i] = padding
	}
	return Canvas{width: width, height: height, buffer: buf}
}

// Width возвращает ширину канваса
func (c Canvas) Width() int { return c.width }

// Height возвращает высоту канваса
func (c Canvas) Height() int { return c.height }

// Get возвращает символ по абсолютным координатам
func (c Canvas) Get(x, y int) rune {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return ' '
	}
	return c.buffer[y*c.width+x]
}

// Set устанавливает символ по абсолютным координатам (для внутреннего использования)
func (c *Canvas) Set(x, y int, ch rune) {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return
	}
	c.buffer[y*c.width+x] = ch
}

// BlitString копирует строку в канвас начиная с позиции (x, y)
// Строка обрезается по правому краю канваса
func (c *Canvas) BlitString(x, y int, s string) {
	runes := []rune(s)
	for i, ch := range runes {
		if x+i >= c.width {
			break
		}
		c.Set(x+i, y, ch)
	}
}

// String возвращает полное текстовое представление канваса
// Все строки имеют одинаковую длину, разделены \n
func (c Canvas) String() string {
	if c.height == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow((c.width + 1) * c.height) // +1 для \n
	for y := 0; y < c.height; y++ {
		start := y * c.width
		end := start + c.width
		sb.WriteString(string(c.buffer[start:end]))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Viewport возвращает срез канваса, готовый к выводу
func (c Canvas) Viewport(x, y, w, h int) string {
	// Нормализация координат
	x0 := clamp(x, 0, c.width)
	y0 := clamp(y, 0, c.height)
	x1 := clamp(x+w, 0, c.width)
	y1 := clamp(y+h, 0, c.height)

	if x1 <= x0 || y1 <= y0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow((x1 - x0 + 1) * (y1 - y0))
	for row := y0; row < y1; row++ {
		start := row*c.width + x0
		end := start + (x1 - x0)
		sb.WriteString(string(c.buffer[start:end]))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// runewidth — упрощённая версия: для примера считаем все руны за 1
// В продакшене использовать: github.com/mattn/go-runewidth
func runeWidth(r rune) int {
	if r < 0x1100 {
		return 1
	}
	// Упрощение: для демо считаем всё за 1
	// Реальная реализация должна учитывать EastAsianWidth
	return 1
}

// stringWidth возвращает ширину строки в экранных позициях
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}
