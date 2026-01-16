local M = {}

function M.usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua capture [options]",
    "",
    "options:",
    "  --scenario PATH    Lua scenario file to run (repeatable)",
    "  --out-dir PATH     Output directory for snapshot files",
    "  --json             Write snapshot.json (default)",
    "  --ansi             Write snapshot.ansi",
    "  --html             Write snapshot.html",
    "  --width N          UI columns (default: 80)",
    "  --height N         UI lines (default: 24)",
    "  --data-home PATH   XDG_DATA_HOME for embedded nvim",
    "  --config-home PATH XDG_CONFIG_HOME for embedded nvim",
    "  --wait MS          Wait for redraw flush (default: 200)",
    "  --rpc-timeout MS   RPC request timeout (default: 2000)",
    "  --nvim PATH        Embedded nvim path (default: nvim)",
    "  --multigrid        Enable ext_multigrid",
    "  -h, --help         Show this help",
  }, "\n")
end

function M.parse(args)
  local opts = {
    scenarios = {},
    out_dir = nil,
    json = false,
    ansi = false,
    html = false,
    wait = 200,
    width = 80,
    height = 24,
    nvim = "nvim",
    rpc_timeout = 2000,
    data_home = nil,
    config_home = nil,
    multigrid = false,
    _flags = {
      output = false,
    },
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--scenario" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--scenario requires a value")
      else
        table.insert(opts.scenarios, value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--scenario=") then
      table.insert(opts.scenarios, string.sub(arg, 12))
    elseif arg == "--out-dir" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--out-dir requires a value")
      else
        opts.out_dir = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--out-dir=") then
      opts.out_dir = string.sub(arg, 11)
    elseif arg == "--json" then
      opts.json = true
      opts._flags.output = true
    elseif arg == "--ansi" then
      opts.ansi = true
      opts._flags.output = true
    elseif arg == "--html" then
      opts.html = true
      opts._flags.output = true
    elseif arg == "--width" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--width requires a value")
      else
        opts.width = tonumber(value) or opts.width
        i = i + 1
      end
    elseif vim.startswith(arg, "--width=") then
      opts.width = tonumber(string.sub(arg, 9)) or opts.width
    elseif arg == "--height" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--height requires a value")
      else
        opts.height = tonumber(value) or opts.height
        i = i + 1
      end
    elseif vim.startswith(arg, "--height=") then
      opts.height = tonumber(string.sub(arg, 10)) or opts.height
    elseif arg == "--data-home" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--data-home requires a value")
      else
        opts.data_home = vim.fn.fnamemodify(value, ":p")
        i = i + 1
      end
    elseif vim.startswith(arg, "--data-home=") then
      opts.data_home = vim.fn.fnamemodify(string.sub(arg, 12), ":p")
    elseif arg == "--config-home" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--config-home requires a value")
      else
        opts.config_home = vim.fn.fnamemodify(value, ":p")
        i = i + 1
      end
    elseif vim.startswith(arg, "--config-home=") then
      opts.config_home = vim.fn.fnamemodify(string.sub(arg, 14), ":p")
    elseif arg == "--wait" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--wait requires a value")
      else
        opts.wait = tonumber(value) or opts.wait
        i = i + 1
      end
    elseif vim.startswith(arg, "--wait=") then
      opts.wait = tonumber(string.sub(arg, 8)) or opts.wait
    elseif arg == "--rpc-timeout" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--rpc-timeout requires a value")
      else
        opts.rpc_timeout = tonumber(value) or opts.rpc_timeout
        i = i + 1
      end
    elseif vim.startswith(arg, "--rpc-timeout=") then
      opts.rpc_timeout = tonumber(string.sub(arg, 15)) or opts.rpc_timeout
    elseif arg == "--nvim" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--nvim requires a value")
      else
        opts.nvim = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--nvim=") then
      opts.nvim = string.sub(arg, 8)
    elseif arg == "--multigrid" then
      opts.multigrid = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  if not opts._flags.output then
    opts.json = true
  end
  return opts
end

return M
