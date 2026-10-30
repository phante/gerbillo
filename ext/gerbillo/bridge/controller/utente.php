<?php
/**
 * Controller del bridge (ADR 0008): risponde al Backend {v, authenticated, user_id, username, groups}.
 * Solo da 127.0.0.1.
 *
 * Non usa $user->data. Il Backend chiama da 127.0.0.1, quindi session_begin() di app.php non
 * riconosce la sessione del browser, che ha un altro IP e un altro user agent, e apre una
 * sessione da ospite. Il controller legge la sessione del cookie _sid dalla tabella, senza
 * crearne né aggiornarne nessuna.
 */

namespace gerbillo\bridge\controller;

use Symfony\Component\HttpFoundation\JsonResponse;

class utente
{
	const VERSIONE = 1;

	protected $config;
	protected $db;
	protected $request;
	protected $user;

	public function __construct($config, $db, $request, $user)
	{
		$this->config = $config;
		$this->db = $db;
		$this->request = $request;
		$this->user = $user;
	}

	public function handle()
	{
		if ($this->request->server('REMOTE_ADDR') !== '127.0.0.1')
		{
			return new JsonResponse(['errore' => 'solo da 127.0.0.1'], 403);
		}

		$utente = $this->utente_della_sessione();
		if ($utente === null)
		{
			return new JsonResponse(['v' => self::VERSIONE, 'authenticated' => false, 'user_id' => ANONYMOUS, 'username' => '', 'groups' => []]);
		}

		return new JsonResponse([
			'v'				=> self::VERSIONE,
			'authenticated'	=> true,
			'user_id'		=> (int) $utente['user_id'],
			'username'		=> $utente['username'],
			'groups'		=> $this->gruppi((int) $utente['user_id']),
		]);
	}

	/**
	 * L'utente della sessione del cookie _sid, se la sessione è valida con le regole di
	 * session_begin(), esclusi IP e user agent. Altrimenti null.
	 */
	protected function utente_della_sessione()
	{
		$sid = $this->request->variable($this->config['cookie_name'] . '_sid', '', false, \phpbb\request\request_interface::COOKIE);
		if ($sid === '')
		{
			return null;
		}

		$sql = 'SELECT s.session_time, s.session_autologin, u.user_id, u.username, u.user_type
			FROM ' . SESSIONS_TABLE . ' s
			JOIN ' . USERS_TABLE . " u ON u.user_id = s.session_user_id
			WHERE s.session_id = '" . $this->db->sql_escape($sid) . "'";
		$result = $this->db->sql_query($sql);
		$riga = $this->db->sql_fetchrow($result);
		$this->db->sql_freeresult($result);

		if (!$riga || (int) $riga['user_id'] === ANONYMOUS || !in_array((int) $riga['user_type'], [USER_NORMAL, USER_FOUNDER], true))
		{
			return null;
		}
		if ($this->scaduta($riga) || $this->user->check_ban((int) $riga['user_id'], false, false, true))
		{
			return null;
		}
		return $riga;
	}

	protected function scaduta(array $riga)
	{
		$ora = time();
		if (!$riga['session_autologin'])
		{
			return $riga['session_time'] < $ora - ($this->config['session_length'] + 60);
		}
		return !$this->config['allow_autologin']
			|| ($this->config['max_autologin_time'] && $riga['session_time'] < $ora - (86400 * (int) $this->config['max_autologin_time']) + 60);
	}

	protected function gruppi($user_id)
	{
		$sql = 'SELECT g.group_name
			FROM ' . USER_GROUP_TABLE . ' ug
			JOIN ' . GROUPS_TABLE . ' g ON g.group_id = ug.group_id
			WHERE ug.user_id = ' . (int) $user_id . '
				AND ug.user_pending = 0';
		$result = $this->db->sql_query($sql);
		$gruppi = [];
		while ($riga = $this->db->sql_fetchrow($result))
		{
			$gruppi[] = $riga['group_name'];
		}
		$this->db->sql_freeresult($result);
		return $gruppi;
	}
}
