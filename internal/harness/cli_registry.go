package harness

import (
	"flag"
	"io"
)

func cmdWorldInit(args []string) int {
	c, remaining, err := parseCommon("world.init", args)
	if err != nil {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if len(remaining) != 0 {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", "unexpected arguments", remaining))
	}
	if c.root == "" || c.worldID == "" {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", "--root and --world-id are required", nil))
	}
	ctx, err := initWorld(c.root, c.worldID)
	if err != nil {
		return emit(failEnvelope("world.init", nil, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("world.init", ctx, nil, map[string]any{"world_id": ctx.ID, "root": ctx.Root}, nil, nil))
}

func cmdWorldStatus(_ commonFlags, ctx *WorldContext, _ []string) int {
	return emit(okEnvelope("world.status", ctx, nil, worldStatus(ctx), nil, nil))
}

func cmdWorldList(args []string) int {
	c, remaining, err := parseCommon("world.list", args)
	if err != nil {
		return emit(failEnvelope("world.list", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world != "" || c.root != "" || len(remaining) != 0 {
		return emit(failEnvelope("world.list", nil, "INVALID_ARGUMENT", "world list does not accept --world, --root, or positional arguments", nil))
	}
	reg, regPath, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("world.list", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	return emit(okEnvelope("world.list", nil, nil, map[string]any{"registry_path": regPath, "worlds": registryWorldList(reg)}, nil, nil))
}

func cmdRegistryList(args []string) int {
	c, remaining, err := parseCommon("registry.list", args)
	if err != nil {
		return emit(failEnvelope("registry.list", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if len(remaining) != 0 {
		return emit(failEnvelope("registry.list", nil, "INVALID_ARGUMENT", "unexpected arguments", remaining))
	}
	reg, regPath, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.list", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	return emit(okEnvelope("registry.list", nil, nil, map[string]any{"registry_path": regPath, "default": reg.Default, "worlds": registryWorldList(reg)}, nil, nil))
}

func cmdRegistryAdd(args []string) int {
	c, remaining, err := parseCommon("registry.add", args)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	fs := flag.NewFlagSet("registry.add.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	title := fs.String("title", "", "title")
	root := fs.String("root", c.root, "root")
	if err := fs.Parse(remaining); err != nil {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || *root == "" || *title == "" {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", "--world, --root, and --title are required", nil))
	}
	abs, err := normalizeRoot(*root, true)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "WORLD_NOT_FOUND", err.Error(), nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	reg.Worlds[c.world] = RegistryWorld{Title: *title, Root: abs}
	if reg.Default == "" {
		reg.Default = c.world
	}
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.add", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath, "registry_root": abs}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}

func cmdRegistryRemove(args []string) int {
	c, remaining, err := parseCommon("registry.remove", args)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || len(remaining) != 0 {
		return emit(failEnvelope("registry.remove", nil, "INVALID_ARGUMENT", "--world is required", nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	delete(reg.Worlds, c.world)
	if reg.Default == c.world {
		reg.Default = ""
	}
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.remove", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}

func cmdRegistryDefault(args []string) int {
	c, remaining, err := parseCommon("registry.default", args)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || len(remaining) != 0 {
		return emit(failEnvelope("registry.default", nil, "INVALID_ARGUMENT", "--world is required", nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	if _, ok := reg.Worlds[c.world]; !ok {
		return emit(failEnvelope("registry.default", nil, "WORLD_NOT_FOUND", "world is not registered", nil))
	}
	reg.Default = c.world
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.default", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}
