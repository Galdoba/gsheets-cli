package render

import "os"

// SaveToFile записывает полный канвас в текстовый файл
// Использует каноническое представление: все строки одинаковой длины, разделены \n
func SaveToFile(c Canvas, filepath string) error {
	content := c.String()
	return os.WriteFile(filepath, []byte(content), 0644)
}
