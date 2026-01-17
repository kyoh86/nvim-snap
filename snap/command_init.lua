local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim-snap init [options]",
    "",
    "options:",
    "  --path PATH      Output workflow path (default: .github/workflows/nvim-snap.yml)",
    "  --root PATH       Root directory for cases (default: .)",
    "  --cases-dir PATH  Cases directory under root (default: snapcase)",
    "  --format FMT     Compare formats (default: html)",
    "  --name NAME      Workflow name (default: nvim-snap)",
    "  --force          Overwrite existing workflow file",
    "  -h, --help       Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    path = ".github/workflows/nvim-snap.yml",
    root = ".",
    cases_dir = "snapcase",
    format = "html",
    name = "nvim-snap",
    force = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--path" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--path requires a value")
      else
        opts.path = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--path=") then
      opts.path = string.sub(arg, 8)
    elseif arg == "--root" then
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
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.format = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.format = string.sub(arg, 10)
    elseif arg == "--name" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--name requires a value")
      else
        opts.name = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--name=") then
      opts.name = string.sub(arg, 8)
    elseif arg == "--force" then
      opts.force = true
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

---@param opts table
---@return string
local function workflow_yaml(opts)
  return table.concat({
    "name: " .. opts.name,
    "",
    "on:",
    "  push:",
    "  pull_request:",
    "",
    "jobs:",
    "  snap:",
    "    runs-on: ubuntu-latest",
    "    steps:",
    "      - uses: actions/checkout@v4",
    "      - name: Install Neovim",
    "        run: sudo apt-get update && sudo apt-get install -y neovim",
    "      - name: Install nvim-snap",
    "        run: |",
    "          curl -sSL https://github.com/kyoh86/nvim-snap/releases/latest/download/nvim-snap -o nvim-snap",
    "          chmod +x nvim-snap",
    "      - name: Run snapshots",
    "        run: |",
    "          ./nvim-snap run --root " .. opts.root
      .. " --cases-dir " .. opts.cases_dir .. " --format json",
    "      - name: Compare snapshots",
    "        run: |",
    "          ./nvim-snap compare --root " .. opts.root
      .. " --cases-dir " .. opts.cases_dir .. " --format " .. opts.format .. " --diff-always",
    "      - name: Upload diffs",
    "        if: always()",
    "        uses: actions/upload-artifact@v4",
    "        with:",
    "          name: nvim-snap-diff",
    "          path: |",
    "            **/diff/*",
    "",
  }, "\n")
end

local function write_file(path, contents, force)
  if vim.fn.filereadable(path) == 1 and not force then
    return nil, "file exists: " .. path
  end
  local dir = vim.fn.fnamemodify(path, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

---@param args_list string[]
function M.run(args_list)
  local opts = parse_args(args_list or {})
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

  local path = vim.fs.normalize(opts.path)
  local ok, err = write_file(path, workflow_yaml(opts), opts.force)
  if not ok then
    util.err_write(err or "failed to write workflow")
    vim.cmd("cq")
    return
  end
  print(path)
end

return M
