local case_def = require("snap.case_def")
local normalize = require("snap.normalize")
local output = require("snap.output")
local png = require("snap.png")
local render = require("snap.render")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim-snap compare [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --cases-dir PATH  Cases directory under root (default: snapcase)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case NAME       Filter by case name (repeatable, comma-separated)",
    "  --format FMT      Diff formats: text,ansi,html,png (default: text)",
    "  --diff-always     Always generate diffs",
    "  --json            Output JSON summary",
    "  -h, --help        Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_format(value)
  local formats = {}
  for _, item in ipairs(split_values(value)) do
    formats[item] = true
  end
  if not next(formats) then
    formats.text = true
  end
  return formats
end

local function parse_args(args)
  local opts = {
    root = ".",
    cases_dir = "snapcase",
    tags = {},
    cases = {},
    formats = { text = true },
    diff_always = false,
    json = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--cases-dir" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--cases-dir requires a value")
      else
        opts.cases_dir = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--cases-dir=") then
      opts.cases_dir = string.sub(arg, 13)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.formats = parse_format(value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.formats = parse_format(string.sub(arg, 10))
    elseif arg == "--diff-always" then
      opts.diff_always = true
    elseif arg == "--json" then
      opts.json = true
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

local function lines_from_matrix(rows, matrix)
  local lines = {}
  for r = 1, rows do
    local row = matrix[r] or {}
    lines[r] = table.concat(row, "")
  end
  return lines
end

local function align_lines(expected_lines, actual_lines)
  local expected_text = table.concat(expected_lines, "\n")
  local actual_text = table.concat(actual_lines, "\n")
  local diffs = vim.text.diff(expected_text, actual_text, {
    result_type = "indices",
    algorithm = "patience",
    linematch = true,
    indent_heuristic = true,
  })
  if type(diffs) ~= "table" then
    diffs = {}
  end
  ---@cast diffs integer[][]
  local pairs = {}
  local e = 1
  local a = 1
  for _, d in ipairs(diffs) do
    local a_start, a_count, b_start, b_count = d[1], d[2], d[3], d[4]
    local expected_unchanged = (a_count == 0) and (a_start - e + 1) or (a_start - e)
    local actual_unchanged = (b_count == 0) and (b_start - a + 1) or (b_start - a)
    if expected_unchanged < 0 then
      expected_unchanged = 0
    end
    if actual_unchanged < 0 then
      actual_unchanged = 0
    end
    local common = math.min(expected_unchanged, actual_unchanged)
    for i = 0, common - 1 do
      table.insert(pairs, { e = e + i, a = a + i, kind = nil })
    end
    if expected_unchanged > common then
      for i = 0, expected_unchanged - common - 1 do
        table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
      end
    elseif actual_unchanged > common then
      for i = 0, actual_unchanged - common - 1 do
        table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
      end
    end
    local e_change = (a_count == 0) and (a_start + 1) or a_start
    local a_change = (b_count == 0) and (b_start + 1) or b_start
    e = e_change
    a = a_change
    local maxc = math.max(a_count, b_count)
    for i = 0, maxc - 1 do
      local er = (i < a_count) and (e + i) or 0
      local ar = (i < b_count) and (a + i) or 0
      local kind = nil
      if er == 0 and ar > 0 then
        kind = "added"
      elseif ar == 0 and er > 0 then
        kind = "removed"
      else
        kind = "changed"
      end
      table.insert(pairs, { e = er, a = ar, kind = kind })
    end
    e = e + a_count
    a = a + b_count
  end
  local expected_remaining = math.max(#expected_lines - e + 1, 0)
  local actual_remaining = math.max(#actual_lines - a + 1, 0)
  local common = math.min(expected_remaining, actual_remaining)
  for i = 0, common - 1 do
    table.insert(pairs, { e = e + i, a = a + i, kind = nil })
  end
  if expected_remaining > common then
    for i = 0, expected_remaining - common - 1 do
      table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
    end
  elseif actual_remaining > common then
    for i = 0, actual_remaining - common - 1 do
      table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
    end
  end
  return pairs
end

local function build_aligned_maps(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local expected_lines = lines_from_matrix(erows, ematrix)
  local actual_lines = lines_from_matrix(arows, amatrix)
  local pairs = align_lines(expected_lines, actual_lines)
  local cols = math.max(ecols, acols)
  local expected_rows = {}
  local actual_rows = {}
  local expected_line_kinds = {}
  local actual_line_kinds = {}
  local expected_cells = {}
  local actual_cells = {}
  for idx, pair in ipairs(pairs) do
    expected_rows[idx] = pair.e
    actual_rows[idx] = pair.a
    if pair.kind == "removed" then
      expected_line_kinds[idx] = "removed"
    elseif pair.kind == "added" then
      actual_line_kinds[idx] = "added"
    elseif pair.kind == "changed" then
      expected_line_kinds[idx] = "removed"
      actual_line_kinds[idx] = "added"
    end
    if pair.e and pair.e > 0 and pair.a and pair.a > 0 then
      for c = 1, cols do
        local etext = ematrix[pair.e] and ematrix[pair.e][c] or " "
        local atext = amatrix[pair.a] and amatrix[pair.a][c] or " "
        if etext ~= atext then
          expected_cells[pair.e] = expected_cells[pair.e] or {}
          actual_cells[pair.a] = actual_cells[pair.a] or {}
          expected_cells[pair.e][c] = true
          actual_cells[pair.a][c] = true
        end
      end
    end
  end
  return {
    expected_rows = expected_rows,
    actual_rows = actual_rows,
    expected_line_kinds = expected_line_kinds,
    actual_line_kinds = actual_line_kinds,
    expected_cells = expected_cells,
    actual_cells = actual_cells,
  }
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

local function wrap_html_diff(unified_diff, expected_plain, actual_plain, expected_aligned, actual_aligned, default_view)
  if default_view ~= "side" and default_view ~= "overlay" then
    default_view = "unified"
  end
  local unified_checked = default_view == "unified" and " checked" or ""
  local side_checked = default_view == "side" and " checked" or ""
  local overlay_checked = default_view == "overlay" and " checked" or ""
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>nvim-snap compare</title>",
    "  <style>",
    "    :root { color-scheme: light; }",
    "    body { margin: 0; background: #f4f5f7; color: #1f2328; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }",
    "    .wrap { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; padding: 0; }",
    "    .panel { background: #ffffff; border: 1px solid #d0d7de; border-radius: 6px; overflow: auto; }",
    "    .title { padding: 8px 12px; border-bottom: 1px solid #222; font-weight: 600; }",
    "    .content { padding: 12px; }",
    "    .content pre { margin: 0; white-space: pre; }",
    "    .section { padding: 12px 16px 16px; }",
    "    .tabs { display: inline-flex; gap: 4px; padding: 10px 12px; margin: 12px 16px 0; align-items: center; background: #ffffff; border: 1px solid #d0d7de; border-radius: 10px; box-shadow: 0 1px 2px rgba(16, 24, 40, 0.08); }",
    "    .tabs .label { color: #57606a; font-size: 1em; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; padding: 4px 8px; border: 1px solid #e1e4e8; border-radius: 999px; background: #f6f8fa; }",
    "    .tabs label { color: #24292f; padding: 6px 12px; border-radius: 8px; border: 1px solid transparent; cursor: pointer; user-select: none; }",
    "    .tabs label:hover { background: #f6f8fa; }",
    "    .toggles { display: none; }",
    "    #view-unified:checked ~ .tabs label[for=\"view-unified\"],",
    "    #view-side:checked ~ .tabs label[for=\"view-side\"],",
    "    #view-side-diff:checked ~ .tabs label[for=\"view-side-diff\"] {",
    "      color: #ffffff; background: #2da44e; border-color: #2da44e; }",
    "    .cell.diff.added { background: rgba(34,197,94,0.28); box-shadow: inset 0 0 0 1px rgba(34,197,94,0.7); }",
    "    .cell.diff.removed { background: rgba(220,38,38,0.22); box-shadow: inset 0 0 0 1px rgba(220,38,38,0.7); }",
    "    .view-plain .line.diff, .view-plain .line.diff .cell, .view-plain .cell.diff { background: none; box-shadow: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.added .cell { box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.removed .cell { box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.added {",
    "      box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.45), inset 0 0 0 1px rgba(34,197,94,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(34,197,94,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.removed {",
    "      box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.4), inset 0 0 0 1px rgba(220,38,38,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(220,38,38,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    .udiff { color: #24292f; }",
    "    .udiff .line { padding: 0 4px; }",
    "    .udiff .line.add { background: rgba(34,197,94,0.18); }",
    "    .udiff .line.del { background: rgba(220,38,38,0.18); }",
    "    .udiff .line.hunk { color: #0969da; }",
    "    #view-unified:not(:checked) ~ .page #unified { display: none; }",
    "    #view-unified:checked ~ .page #side { display: none; }",
    "    #view-side:checked ~ .page #side { display: block; }",
    "    #view-side-diff:checked ~ .page #side { display: block; }",
    "    #view-side:checked ~ .page #side .view-plain { display: block; }",
    "    #view-side:checked ~ .page #side .view-aligned { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-plain { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned { display: block; }",
    "    .grid { display: inline-block; }",
    "    .line { display: block; white-space: pre; }",
    "    .cell { display: inline-block; }",
    "  </style>",
    "</head>",
    "<body>",
    "  <input id=\"view-unified\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. unified_checked .. " />",
    "  <input id=\"view-side\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. side_checked .. " />",
    "  <input id=\"view-side-diff\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. overlay_checked .. " />",
    "  <div class=\"tabs\">",
    "    <span class=\"label\">view</span>",
    "    <label for=\"view-unified\">unified</label>",
    "    <label for=\"view-side\">side</label>",
    "    <label for=\"view-side-diff\">overlay</label>",
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
    "        <div class=\"content view-plain\" style=\"background:" .. expected_plain.bg .. ";color:" .. expected_plain.fg .. ";\"><div class=\"grid\">" .. expected_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. expected_aligned.bg .. ";color:" .. expected_aligned.fg .. ";\"><div class=\"grid\">" .. expected_aligned.html .. "</div></div>",
    "      </div>",
    "      <div class=\"panel\">",
    "        <div class=\"title\">actual</div>",
    "        <div class=\"content view-plain\" style=\"background:" .. actual_plain.bg .. ";color:" .. actual_plain.fg .. ";\"><div class=\"grid\">" .. actual_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. actual_aligned.bg .. ";color:" .. actual_aligned.fg .. ";\"><div class=\"grid\">" .. actual_aligned.html .. "</div></div>",
    "      </div>",
    "    </div>",
    "  </div>",
    "  </div>",
    "</body>",
    "</html>",
  }, "\n")
end

local function render_html_diff(expected, actual, default_view)
  local expected_render_text = render.render_text(expected)
  local actual_render_text = render.render_text(actual)
  local unified = vim.text.diff(expected_render_text, actual_render_text, { result_type = "unified", ctxlen = 3 })
  local diff_map = build_diff_map(expected, actual)
  local expected_plain = render.render_html_cells(expected, diff_map.expected, "removed")
  local actual_plain = render.render_html_cells(actual, diff_map.actual, "added")
  local aligned = build_aligned_maps(expected, actual)
  local expected_aligned = render.render_html_aligned(
    expected,
    aligned.expected_rows,
    aligned.expected_line_kinds,
    aligned.expected_cells,
    "removed"
  )
  local actual_aligned = render.render_html_aligned(
    actual,
    aligned.actual_rows,
    aligned.actual_line_kinds,
    aligned.actual_cells,
    "added"
  )
  return wrap_html_diff(
    highlight_unified(unified or ""),
    expected_plain,
    actual_plain,
    expected_aligned,
    actual_aligned,
    default_view
  )
end

local function diff_files(expected, actual, formats)
  local results = {}
  if formats.text then
    local diff = vim.text.diff(render.render_text(expected), render.render_text(actual), {
      result_type = "unified",
      ctxlen = 3,
    })
    results.text = diff or ""
  end
  if formats.ansi then
    local diff = vim.text.diff(render.render_ansi(expected), render.render_ansi(actual), {
      result_type = "unified",
      ctxlen = 3,
    })
    results.ansi = diff or ""
  end
  if formats.html then
    results.html = render_html_diff(expected, actual, "unified")
  end
  if formats.png then
    results.png = render_html_diff(expected, actual, "overlay")
  end
  return results
end

local function write_diff_outputs(c, outputs)
  local ok, err = case_def.ensure_dir(c.diff_dir)
  if not ok then
    return nil, err
  end
  local diff_paths = {}
  if outputs.text ~= nil then
    local path = util.joinpath(c.diff_dir, "diff.txt")
    local ok_write, write_err = output.write(path, outputs.text)
    if not ok_write then
      return nil, write_err or "failed to write diff.txt"
    end
    diff_paths.text = path
  end
  if outputs.ansi ~= nil then
    local path = util.joinpath(c.diff_dir, "diff.ansi")
    local ok_write, write_err = output.write(path, outputs.ansi)
    if not ok_write then
      return nil, write_err or "failed to write diff.ansi"
    end
    diff_paths.ansi = path
  end
  if outputs.html ~= nil then
    local path = util.joinpath(c.diff_dir, "diff.html")
    local ok_write, write_err = output.write(path, outputs.html)
    if not ok_write then
      return nil, write_err or "failed to write diff.html"
    end
    diff_paths.html = path
  end
  if outputs.png ~= nil then
    local path = util.joinpath(c.diff_dir, "diff.png")
    local ok_write, write_err = png.write_png_from_html(outputs.png, path)
    if not ok_write then
      return nil, write_err or "failed to write diff.png"
    end
    diff_paths.png = path
  end
  return diff_paths
end

local function print_text(results)
  for _, r in ipairs(results) do
    local tags = table.concat(r.tags, ",")
    local diff_paths = ""
    if r.diff_paths then
      local items = {}
      for key, value in pairs(r.diff_paths) do
        table.insert(items, key .. "=" .. value)
      end
      table.sort(items)
      diff_paths = table.concat(items, ",")
    end
    print(table.concat({
      r.name,
      r.title,
      r.kind,
      tags,
      r.result,
      diff_paths,
      r.error_reason or "",
    }, "\t"))
  end
end

local function print_json(root, results, summary)
  local out = {
    root = root,
    summary = summary,
    cases = results,
  }
  print(vim.json.encode(out, { indent = "  " }))
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

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases, errors = case_def.find_cases(root, opts.cases_dir)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })
  local results = {}
  local summary = { total = 0, no_diff = 0, diff = 0, error = 0 }
  local has_error = #errors > 0
  local has_diff = false

  for _, err in ipairs(errors) do
    util.err_write(err)
  end

  for _, c in ipairs(filtered) do
    summary.total = summary.total + 1
    local entry = {
      name = c.name,
      title = c.title,
      kind = c.kind,
      tags = c.tags,
      result = "error",
      diff_paths = nil,
      error_reason = nil,
    }
    local expected_text, expected_err = read_input(c.expected_path)
    if not expected_text then
      entry.error_reason = expected_err or "expected not found"
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.name .. ": " .. entry.error_reason)
      goto continue
    end
    local actual_text, actual_err = read_input(c.actual_path)
    if not actual_text then
      entry.error_reason = actual_err or "actual not found"
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.name .. ": " .. entry.error_reason)
      goto continue
    end
    local expected_snapshot, expected_decode_err = decode_json(expected_text)
    if not expected_snapshot then
      entry.error_reason = expected_decode_err
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.name .. ": " .. entry.error_reason)
      goto continue
    end
    local actual_snapshot, actual_decode_err = decode_json(actual_text)
    if not actual_snapshot then
      entry.error_reason = actual_decode_err
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.name .. ": " .. entry.error_reason)
      goto continue
    end

    local normalized_expected = normalize.normalize(expected_snapshot)
    local normalized_actual = normalize.normalize(actual_snapshot)
    local equal = deep_equal(normalized_expected, normalized_actual)
    if equal then
      entry.result = "no_diff"
      summary.no_diff = summary.no_diff + 1
    else
      entry.result = "diff"
      summary.diff = summary.diff + 1
      has_diff = true
    end

    if entry.result == "diff" or opts.diff_always then
      local outputs = diff_files(normalized_expected, normalized_actual, opts.formats)
      local diff_paths, write_err = write_diff_outputs(c, outputs)
      if not diff_paths then
        entry.result = "error"
        entry.error_reason = write_err or "failed to write diff"
        summary.error = summary.error + 1
        has_error = true
        util.err_write(c.name .. ": " .. entry.error_reason)
      else
        entry.diff_paths = diff_paths
      end
    end

    table.insert(results, entry)
    ::continue::
  end

  if opts.json then
    print_json(root, results, summary)
  else
    print_text(results)
  end

  if has_error then
    vim.cmd("cquit 2")
  elseif has_diff then
    vim.cmd("cquit 1")
  end
end

return M
