package utils

// This file contains utility functions for styling components in pure Go
// to avoid templ import issues

// GetColorClasses gibt CSS-Klassen basierend auf einem Farbschema zurück
func GetColorClasses(variant string, component string) string {
	baseClass := ""

	// Komponenten-spezifische Basisklassen
	switch component {
	case "button":
		baseClass = "font-medium rounded-lg focus:outline-none focus:ring-2 focus:ring-offset-2 "
	case "badge":
		baseClass = "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "
	case "alert":
		baseClass = "p-4 rounded-lg border-l-4 "
	case "dropdown-item":
		baseClass = "flex items-center px-4 py-2 text-sm "
	}

	// Farb-spezifische Klassen
	switch variant {
	case "primary", "brand":
		return baseClass + "bg-brand-600 hover:bg-brand-700 text-white focus:ring-brand-500"
	case "secondary":
		return baseClass + "bg-gray-200 hover:bg-gray-300 text-gray-700 focus:ring-gray-500"
	case "success", "green":
		return baseClass + "bg-green-600 hover:bg-green-700 text-white focus:ring-green-500"
	case "danger", "red":
		return baseClass + "bg-red-600 hover:bg-red-700 text-white focus:ring-red-500"
	case "warning", "yellow":
		return baseClass + "bg-yellow-500 hover:bg-yellow-600 text-white focus:ring-yellow-500"
	case "info", "blue":
		return baseClass + "bg-blue-600 hover:bg-blue-700 text-white focus:ring-blue-500"
	default:
		return baseClass + "bg-gray-200 hover:bg-gray-300 text-gray-700 focus:ring-gray-500"
	}
}

// GetSizeClasses gibt CSS-Klassen basierend auf einer Größe zurück
func GetSizeClasses(size string, component string) string {
	switch component {
	case "button":
		switch size {
		case "sm", "small":
			return "px-3 py-1.5 text-sm"
		case "lg", "large":
			return "px-6 py-3 text-lg"
		default: // Medium ist der Standard
			return "px-4 py-2 text-base"
		}
	case "input":
		switch size {
		case "sm", "small":
			return "px-3 py-1.5 text-sm"
		case "lg", "large":
			return "px-5 py-3.5 text-lg"
		default:
			return "px-4 py-3 text-base"
		}
	default:
		return ""
	}
}
