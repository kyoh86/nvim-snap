local M = {}

function M.usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua capture [options]",
    "",
    "options:",
    "  --case PATH        Run case directory (snapcase.json + scenario.lua)",
    "  --json-out PATH    JSON output path ('-' for stdout, 'none' to skip)",
    "  --ansi-out PATH    Write ANSI preview ('-' for stdout)",
    "  --html-out PATH    Write HTML preview ('-' for stdout)",
    "  --width N          UI columns (default: 80)",
    "  --height N         UI lines (default: 24)",
    "  --script PATH      Execute Lua scenario file before snapshot (repeatable)",
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
    scripts = {},
    wait = 200,
    json_out = "-",
    width = 80,
    height = 24,
    nvim = "nvim",
    rpc_timeout = 2000,
    ansi_out = nil,
    html_out = nil,
    data_home = nil,
    config_home = nil,
    case = nil,
    _flags = {
      json_out = false,
      ansi_out = false,
      html_out = false,
      script = false,
      case = false,
    },
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        opts.case = value
        opts._flags.case = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      opts.case = string.sub(arg, 8)
      opts._flags.case = true
    elseif arg == "--out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--out requires a value")
      else
        opts.json_out = value
        opts._flags.json_out = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--out=") then
      opts.json_out = string.sub(arg, 7)
      opts._flags.json_out = true
    elseif arg == "--json-out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--json-out requires a value")
      else
        opts.json_out = value
        opts._flags.json_out = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--json-out=") then
      opts.json_out = string.sub(arg, 12)
      opts._flags.json_out = true
    elseif arg == "--ansi-out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--ansi-out requires a value")
      else
        opts.ansi_out = value
        opts._flags.ansi_out = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--ansi-out=") then
      opts.ansi_out = string.sub(arg, 12)
      opts._flags.ansi_out = true
    elseif arg == "--html-out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--html-out requires a value")
      else
        opts.html_out = value
        opts._flags.html_out = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--html-out=") then
      opts.html_out = string.sub(arg, 12)
      opts._flags.html_out = true
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
    elseif arg == "--script" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--script requires a value")
      else
        table.insert(opts.scripts, value)
        opts._flags.script = true
        i = i + 1
      end
    elseif vim.startswith(arg, "--script=") then
      table.insert(opts.scripts, string.sub(arg, 10))
      opts._flags.script = true
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
  return opts
end

return M
