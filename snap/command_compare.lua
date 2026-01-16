local normalize = require("snap.normalize")
local output = require("snap.output")
local render = require("snap.render")
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
    "  --diff-format FMT  Diff source: text|ansi|html (default: text)",
    "  --diff-out PATH    Write diff to PATH (default: stdout)",
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
    diff_format = "text",
    diff_out = "-",
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
    elseif arg == "--diff-format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--diff-format requires a value")
      else
        opts.diff_format = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--diff-format=") then
      opts.diff_format = string.sub(arg, 15)
    elseif arg == "--diff-out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--diff-out requires a value")
      else
        opts.diff_out = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--diff-out=") then
      opts.diff_out = string.sub(arg, 12)
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

local function escape_html(text)
  return (text:gsub("[&<>\"']", {
    ["&"] = "&amp;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ['"'] = "&quot;",
    ["'"] = "&#39;",
  }))
end

local function highlight_unified(diff_text)
  local out = { "<pre>" }
  for line in diff_text:gmatch("([^\n]*)\n?") do
    if line == "" and #out > 1 and out[#out] == "" then
      break
    end
    local cls = "line"
    if vim.startswith(line, "+") then
      cls = cls .. " add"
    elseif vim.startswith(line, "-") then
      cls = cls .. " del"
    elseif vim.startswith(line, "@@") then
      cls = cls .. " hunk"
    end
    table.insert(out, '<div class="' .. cls .. '">' .. escape_html(line) .. "</div>")
  end
  table.insert(out, "</pre>")
  return table.concat(out)
end

local function render_for_diff(snapshot, format)
  if format == "ansi" then
    return render.render_ansi(snapshot)
  end
  return render.render_text(snapshot)
end

local function wrap_html_diff(unified_diff, expected_block, actual_block)
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>nvim-snap compare</title>",
    "  <style>",
    "    :root { color-scheme: dark; }",
    "    body { margin: 0; background: #111; color: #eee; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }",
    "    .wrap { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; padding: 16px; }",
    "    .panel { background: #0b0b0b; border: 1px solid #222; border-radius: 6px; overflow: auto; }",
    "    .title { padding: 8px 12px; border-bottom: 1px solid #222; font-weight: 600; }",
    "    .content { padding: 12px; }",
    "    .content pre { margin: 0; white-space: pre; }",
    "    .tabs { display: flex; gap: 12px; padding: 12px 16px 0; align-items: center; flex-wrap: wrap; }",
    "    .tabs label { color: #bbb; text-decoration: none; padding: 4px 8px; border: 1px solid #333; border-radius: 4px; cursor: pointer; }",
    "    .tabs label:hover { color: #fff; border-color: #555; }",
    "    .toggles { position: absolute; opacity: 0; pointer-events: none; }",
    "    #view-unified:checked ~ .tabs label[for=\"view-unified\"],",
    "    #view-side:checked ~ .tabs label[for=\"view-side\"],",
    "    #view-side-diff:checked ~ .tabs label[for=\"view-side-diff\"] {",
      "      color: #fff; border-color: #5a8; background: #143;",
    "    }",
    "    .section { padding: 12px 16px 16px; }",
    "    .line.diff.added { background: rgba(0,200,120,0.28); }",
    "    .line.diff.removed { background: rgba(220,70,70,0.28); }",
    "    .cell.diff.added { background: rgba(0,200,120,0.55); box-shadow: inset 0 0 0 1px rgba(0,200,120,0.8); }",
    "    .cell.diff.removed { background: rgba(220,70,70,0.55); box-shadow: inset 0 0 0 1px rgba(220,70,70,0.8); }",
    "    .view .line.diff, .view .cell.diff { background: none; box-shadow: none; }",
    "    #view-side-diff:checked ~ .page .view .line.diff.added { background: rgba(0,200,120,0.28); }",
    "    #view-side-diff:checked ~ .page .view .line.diff.removed { background: rgba(220,70,70,0.28); }",
    "    #view-side-diff:checked ~ .page .view .cell.diff.added { background: rgba(0,200,120,0.55); box-shadow: inset 0 0 0 1px rgba(0,200,120,0.8); }",
    "    #view-side-diff:checked ~ .page .view .cell.diff.removed { background: rgba(220,70,70,0.55); box-shadow: inset 0 0 0 1px rgba(220,70,70,0.8); }",
    "    .udiff { color: #ddd; }",
    "    .udiff .line { padding: 0 4px; }",
    "    .udiff .line.add { background: rgba(0,200,120,0.22); }",
    "    .udiff .line.del { background: rgba(220,70,70,0.22); }",
    "    .udiff .line.hunk { color: #9ad; }",
    "    #view-unified:not(:checked) ~ .page #unified { display: none; }",
    "    #view-side:not(:checked) ~ .page #side { display: none; }",
    "    #view-side-diff:checked ~ .page #side { display: block; }",
    "    .grid { display: inline-block; }",
    "    .line { display: block; }",
    "    .cell { display: inline-block; }",
    "  </style>",
    "</head>",
    "<body>",
    "  <input id=\"view-unified\" class=\"toggles\" type=\"radio\" name=\"diff-view\" checked />",
    "  <input id=\"view-side\" class=\"toggles\" type=\"radio\" name=\"diff-view\" />",
    "  <input id=\"view-side-diff\" class=\"toggles\" type=\"radio\" name=\"diff-view\" />",
    "  <div class=\"tabs\">",
    "    <span>diff view:</span>",
    "    <label for=\"view-unified\">unified</label>",
    "    <label for=\"view-side\">side-by-side</label>",
    "    <label for=\"view-side-diff\">side-by-side + overlay</label>",
    "  </div>",
    "  <div class=\"page\">",
    "  <div class=\"section\" id=\"unified\">",
    "    <div class=\"panel\">",
    "      <div class=\"title\">unified diff (text)</div>",
    "      <div class=\"content udiff\">" .. unified_diff .. "</div>",
    "    </div>",
    "  </div>",
    "  <div class=\"section\" id=\"side\">",
    "    <div class=\"wrap\">",
    "      <div class=\"panel\">",
    "        <div class=\"title\">expected</div>",
    "        <div class=\"content view\" style=\"background:" .. expected_block.bg .. ";color:" .. expected_block.fg .. ";\"><div class=\"grid\">" .. expected_block.html .. "</div></div>",
    "      </div>",
    "      <div class=\"panel\">",
    "        <div class=\"title\">actual</div>",
    "        <div class=\"content view\" style=\"background:" .. actual_block.bg .. ";color:" .. actual_block.fg .. ";\"><div class=\"grid\">" .. actual_block.html .. "</div></div>",
    "      </div>",
    "    </div>",
    "  </div>",
    "  </div>",
    "</body>",
    "</html>",
  }, "\n")
end

local function grid_text_matrix(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return 0, 0, {}
  end
  local rows = grid.rows or 0
  local cols = grid.cols or 0
  local matrix = {}
  for r = 1, rows do
    local row_cells = grid.cells[r] or {}
    local line = {}
    for c = 1, cols do
      local cell = row_cells[c] or { text = " " }
      local text = cell.text
      if text == "" then
        text = " "
      end
      line[c] = text
    end
    matrix[r] = line
  end
  return rows, cols, matrix
end

local function build_diff_map(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local rows = math.max(erows, arows)
  local cols = math.max(ecols, acols)
  local expected = { lines = {}, cells = {} }
  local actual = { lines = {}, cells = {} }
  for r = 1, rows do
    local line_diff = false
    for c = 1, cols do
      local etext = ematrix[r] and ematrix[r][c] or " "
      local atext = amatrix[r] and amatrix[r][c] or " "
      if etext ~= atext then
        expected.cells[r] = expected.cells[r] or {}
        actual.cells[r] = actual.cells[r] or {}
        expected.cells[r][c] = true
        actual.cells[r][c] = true
        line_diff = true
      end
    end
    if line_diff then
      expected.lines[r] = true
      actual.lines[r] = true
    end
  end
  return { expected = expected, actual = actual }
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
  if opts.diff_format ~= "text" and opts.diff_format ~= "ansi" and opts.diff_format ~= "html" then
    util.err_write("--diff-format must be text, ansi, or html")
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
    if opts.diff_format == "html" then
      local expected_render_text = render.render_text(normalized_expected)
      local actual_render_text = render.render_text(normalized_actual)
      local unified = vim.text.diff(expected_render_text, actual_render_text, { result_type = "unified", ctxlen = 3 })
      local diff_map = build_diff_map(normalized_expected, normalized_actual)
      local expected_cells = render.render_html_cells(normalized_expected, diff_map.expected, "removed")
      local actual_cells = render.render_html_cells(normalized_actual, diff_map.actual, "added")
      local rendered = wrap_html_diff(
        highlight_unified(unified),
        expected_cells,
        actual_cells
      )
      local ok, write_err = output.write(opts.diff_out, rendered)
      if not ok then
        util.err_write(write_err or "failed to write diff")
        vim.cmd("cq")
        return
      end
    else
      local expected_out = render_for_diff(normalized_expected, opts.diff_format)
      local actual_out = render_for_diff(normalized_actual, opts.diff_format)
      local diff = vim.text.diff(expected_out, actual_out, { result_type = "unified", ctxlen = 3 })
      if diff then
        local ok, write_err = output.write(opts.diff_out, diff)
        if not ok then
          util.err_write(write_err or "failed to write diff")
          vim.cmd("cq")
          return
        end
      end
    end
  end

  util.err_write("mismatch")
  vim.cmd("cq")
end

return M
