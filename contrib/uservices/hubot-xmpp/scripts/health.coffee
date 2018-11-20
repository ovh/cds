
module.exports = (robot) ->
  robot.router.get '/health', (req, res) ->
    res.writeHead 200, {'Content-Type': 'text/plain'}
    res.end "I'm Alive!\n"
