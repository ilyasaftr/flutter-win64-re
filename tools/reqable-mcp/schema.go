package main

func obj(props map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func boolean(desc string, def bool) map[string]any {
	return map[string]any{"type": "boolean", "description": desc, "default": def}
}

func intSchema(desc string, def int) map[string]any {
	return map[string]any{"type": "integer", "description": desc, "default": def}
}

func arr(desc string) map[string]any { return map[string]any{"type": "array", "description": desc} }

func rawObj(desc string) map[string]any { return map[string]any{"type": "object", "description": desc} }
