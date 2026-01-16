local normalize = require("snap.normalize")
local output = require("snap.output")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua compare [options]",
    "",
    "options:",
    "  --actual PATH      Snapshot JSON path ('-' for stdin)",
    "  --expected PATH    Expected JSON path",
    "  --update           Overwrite expected with actual",
    "  --pretty           Pretty-print JSON when updating",
    "  --diff             Print unified diff on mismatch",
    "  -h, --help         Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    actual = nil,
    expected = nil,
    update = false,
    pretty = false,
    diff = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--actual" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--actual requires a value")
      else
        opts.actual = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--actual=") then
      opts.actual = string.sub(arg, 10)
    elseif arg == "--expected" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--expected requires a value")
      else
        opts.expected = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--expected=") then
      opts.expected = string.sub(arg, 12)
    elseif arg == "--update" then
      opts.update = true
    elseif arg == "--pretty" then
      opts.pretty = true
    elseif arg == "--diff" then
      opts.diff = true
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

local function decode_json(text)
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    return nil, "failed to parse json: " .. tostring(decoded)
  end
  return decoded
end

local function deep_equal(a, b)
  if type(a) ~= type(b) then
    return false
  end
  if type(a) ~= "table" then
    return a == b
  end
  local seen = {}
  for key, value in pairs(a) do
    if not deep_equal(value, b[key]) then
      return false
    end
    seen[key] = true
  end
  for key, _ in pairs(b) do
    if not seen[key] then
      return false
    end
  end
  return true
end

local function encode_json(value, pretty)
  if pretty then
    return vim.json.encode(value, { indent = "  " })
  end
  return vim.json.encode(value)
end

local function maybe_diff(expected_text, actual_text)
  if not vim.diff then
    return nil
  end
  local ok, diff = pcall(vim.diff, expected_text, actual_text, { result_type = "unified", ctxlen = 3 })
  if not ok then
    return nil
  end
  return diff
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
  if not opts.actual or opts.actual == "" then
    util.err_write("--actual is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if not opts.expected or opts.expected == "" then
    util.err_write("--expected is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local actual_text, actual_err = read_input(opts.actual)
  if not actual_text then
    util.err_write(actual_err or "failed to read actual")
    vim.cmd("cq")
    return
  end
  local actual_snapshot, actual_decode_err = decode_json(actual_text)
  if not actual_snapshot then
    util.err_write(actual_decode_err or "failed to parse actual")
    vim.cmd("cq")
    return
  end

  local expected_text, expected_err = read_input(opts.expected)
  local expected_snapshot = nil
  if expected_text then
    local decoded, decode_err = decode_json(expected_text)
    if not decoded then
      util.err_write(decode_err or "failed to parse expected")
      vim.cmd("cq")
      return
    end
    expected_snapshot = decoded
  elseif not opts.update then
    util.err_write(expected_err or "failed to read expected")
    vim.cmd("cq")
    return
  end

  local normalized_actual = normalize.normalize(actual_snapshot)
  local normalized_expected = expected_snapshot and normalize.normalize(expected_snapshot) or nil

  if normalized_expected and deep_equal(normalized_actual, normalized_expected) then
    print("match")
    return
  end

  if opts.update then
    local encoded = encode_json(normalized_actual, opts.pretty)
    local ok, write_err = output.write(opts.expected, encoded)
    if not ok then
      util.err_write(write_err or "failed to update expected")
      vim.cmd("cq")
      return
    end
    print("updated")
    return
  end

  if opts.diff and normalized_expected then
    local expected_out = encode_json(normalized_expected, true)
    local actual_out = encode_json(normalized_actual, true)
    local diff = maybe_diff(expected_out, actual_out)
    if diff then
      io.write(diff)
    end
  end

  util.err_write("mismatch")
  vim.cmd("cq")
end

return M
