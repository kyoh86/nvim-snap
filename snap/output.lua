local M = {}

function M.write(out, contents)
  if out == "-" then
    io.write(contents)
    io.write("\n")
    return true
  end
  local dir = vim.fn.fnamemodify(out, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(out, "w")
  if not fd then
    return false, err
  end
  fd:write(contents)
  fd:close()
  return true
end

return M
