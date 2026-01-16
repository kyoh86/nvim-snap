local normalize = require("snap.normalize")
local output = require("snap.output")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua core normalize [options]",
    "",
    "options:",
    "  --in PATH       Input JSON path ('-' for stdin)",
    "  --out PATH      Output JSON path ('-' for stdout)",
    "  --pretty        Pretty-print JSON",
    "  -h, --help      Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    input = "-",
    output = "-",
    pretty = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--in" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--in requires a value")
      else
        opts.input = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--in=") then
      opts.input = string.sub(arg, 6)
    elseif arg == "--out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--out requires a value")
      else
        opts.output = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--out=") then
      opts.output = string.sub(arg, 7)
    elseif arg == "--pretty" then
      opts.pretty = true
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

local function read_input(path)
  if path == "-" then
    return io.read("*a")
  end
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local text, err = read_input(opts.input)
  if not text then
    util.err_write(err or "failed to read input")
    vim.cmd("cq")
    return
  end
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    util.err_write("failed to parse json: " .. tostring(decoded))
    vim.cmd("cq")
    return
  end

  local normalized = normalize.normalize(decoded)
  local encoded = vim.json.encode(normalized)
  if opts.pretty then
    encoded = vim.json.encode(normalized, { indent = "  " })
  end

  local ok_write, write_err = output.write(opts.output, encoded)
  if not ok_write then
    util.err_write(write_err or "failed to write output")
    vim.cmd("cq")
  end
end

return M
