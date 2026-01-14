local M = {}

function M.rpc_error_to_string(err)
  if err == nil or err == vim.NIL then
    return "unknown error"
  end
  if type(err) == "table" then
    return vim.inspect(err)
  end
  return tostring(err)
end

function M.err_write(msg)
  if msg == nil then
    msg = "unknown error"
  end
  io.stderr:write(tostring(msg), "\n")
end

function M.normalize_path(base, path)
  if not path then
    return nil
  end
  if vim.fn.fnamemodify(path, ":p") == path then
    return vim.fs.normalize(path)
  end
  return vim.fs.normalize(vim.fs.joinpath(base, path))
end

return M
