package ui

type Color string

const (
	ColorDefault     Color = "\033[0m"
	ColorBlack       Color = "\033[38;2;0;0;0m"
	ColorDarkGray    Color = "\033[38;2;100;100;100m"
	ColorGray        Color = "\033[38;2;150;150;150m"
	ColorWhite       Color = "\033[38;2;255;255;255m"
	ColorBrightWhite Color = "\033[38;2;255;255;255;1m"

	ColorLightRed Color = "\033[38;2;255;150;150m"
	ColorRed      Color = "\033[38;2;255;0;0m"
	ColorDarkRed  Color = "\033[38;2;150;0;0m"

	ColorLightGreen Color = "\033[38;2;150;255;150m"
	ColorGreen      Color = "\033[38;2;0;255;0m"
	ColorDarkGreen  Color = "\033[38;2;0;150;0m"

	ColorLightYellow Color = "\033[38;2;255;255;150m"
	ColorYellow      Color = "\033[38;2;255;255;0m"
	ColorDarkYellow  Color = "\033[38;2;150;150;0m"

	ColorLightBlue Color = "\033[38;2;150;150;255m"
	ColorBlue      Color = "\033[38;2;0;0;255m"
	ColorDarkBlue  Color = "\033[38;2;0;0;150m"

	ColorLightBrown Color = "\033[38;2;210;180;140m"
	ColorBrown      Color = "\033[38;2;165;42;42m"
	ColorDarkBrown  Color = "\033[38;2;101;67;33m"

	ColorLightPurple Color = "\033[38;2;200;150;255m"
	ColorPurple      Color = "\033[38;2;128;0;128m"
	ColorDarkPurple  Color = "\033[38;2;75;0;130m"

	ColorLightOrange Color = "\033[38;2;255;200;150m"
	ColorOrange      Color = "\033[38;2;255;165;0m"
	ColorDarkOrange  Color = "\033[38;2;255;140;0m"

	ColorPink Color = "\033[38;2;255;192;203m"
)
