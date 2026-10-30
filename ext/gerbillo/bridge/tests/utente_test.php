<?php
// Prova del controller senza phpBB: php ext/gerbillo/bridge/tests/utente_test.php
// La prova sul forum vero è in #44.

namespace Symfony\Component\HttpFoundation {
	class JsonResponse
	{
		public $dati;
		public $codice;
		public function __construct($dati, $codice = 200)
		{
			$this->dati = $dati;
			$this->codice = $codice;
		}
	}
}

namespace phpbb\request {
	interface request_interface
	{
		const COOKIE = 3;
	}
}

namespace {
	const ANONYMOUS = 1, USER_NORMAL = 0, USER_INACTIVE = 1, USER_IGNORE = 2, USER_FOUNDER = 3;
	const SESSIONS_TABLE = 'sessions', USERS_TABLE = 'users', USER_GROUP_TABLE = 'user_group', GROUPS_TABLE = 'groups';

	require __DIR__ . '/../controller/utente.php';

	class richiesta_finta
	{
		public function __construct(public $ip, public $sid) {}
		public function server($nome) { return $nome === 'REMOTE_ADDR' ? $this->ip : ''; }
		public function variable($nome, $predefinito, $multibyte, $dove)
		{
			return $nome === 'phpbb3_sid' && $dove === \phpbb\request\request_interface::COOKIE ? $this->sid : $predefinito;
		}
	}

	class db_finto
	{
		public $query = [];
		private $righe = [];
		public function __construct(public $sessione, public $gruppi = []) {}
		public function sql_escape($s) { return addslashes($s); }
		public function sql_query($sql)
		{
			$this->query[] = $sql;
			$this->righe = str_contains($sql, SESSIONS_TABLE) ? ($this->sessione ? [$this->sessione] : []) : array_map(fn($g) => ['group_name' => $g], $this->gruppi);
			return true;
		}
		public function sql_fetchrow($r) { return array_shift($this->righe) ?: false; }
		public function sql_freeresult($r) {}
	}

	class user_finto
	{
		public function __construct(public $bannato = false) {}
		public function check_ban($id, $ip, $email, $ritorna) { return $this->bannato; }
	}

	function chiama($ip, $sid, $sessione, $bannato = false, $config = [])
	{
		$config += ['cookie_name' => 'phpbb3', 'session_length' => 3600, 'allow_autologin' => 1, 'max_autologin_time' => 0];
		$db = new db_finto($sessione, ['REGISTERED', 'GLOBAL_MODERATORS']);
		$c = new \gerbillo\bridge\controller\utente($config, $db, new richiesta_finta($ip, $sid), new user_finto($bannato));
		return [$c->handle(), $db];
	}

	function verifica($nome, $ok)
	{
		global $falliti;
		if (!$ok)
		{
			$falliti++;
			echo "FALLITO: $nome\n";
		}
	}

	$falliti = 0;
	$valida = ['session_time' => time(), 'session_autologin' => 0, 'user_id' => 8, 'username' => 'mod', 'user_type' => USER_NORMAL];

	// Solo da 127.0.0.1, prima di guardare il database.
	foreach (['8.8.8.8', '127.0.0.2', '::1', '', '127.0.0.1, 8.8.8.8'] as $ip)
	{
		[$r, $db] = chiama($ip, 'abc', $valida);
		verifica("rifiuta l'IP «{$ip}»", $r->codice === 403 && !isset($r->dati['authenticated']) && !$db->query);
	}

	[$r] = chiama('127.0.0.1', 'abc', $valida);
	verifica('sessione valida', $r->codice === 200 && $r->dati === ['v' => 1, 'authenticated' => true, 'user_id' => 8, 'username' => 'mod', 'groups' => ['REGISTERED', 'GLOBAL_MODERATORS']]);

	[$r, $db] = chiama('127.0.0.1', "a'b", $valida);
	verifica('il sid è sottoposto a escape', str_contains($db->query[0], "a\\'b"));

	$ospiti = [
		'senza cookie'			=> ['', $valida],
		'sessione inesistente'	=> ['abc', null],
		'sessione da ospite'	=> ['abc', ['user_id' => ANONYMOUS] + $valida],
		'utente inattivo'		=> ['abc', ['user_type' => USER_INACTIVE] + $valida],
		'bot'					=> ['abc', ['user_type' => USER_IGNORE] + $valida],
		'sessione scaduta'		=> ['abc', ['session_time' => time() - 4000] + $valida],
		'ricordami, autologin spento'	=> ['abc', ['session_autologin' => 1] + $valida, false, ['allow_autologin' => 0]],
		'ricordami, troppo vecchia'		=> ['abc', ['session_autologin' => 1, 'session_time' => time() - 86400 * 3] + $valida, false, ['max_autologin_time' => 1]],
		'utente bannato'		=> ['abc', $valida, true],
	];
	foreach ($ospiti as $nome => $caso)
	{
		[$r] = chiama('127.0.0.1', ...$caso);
		verifica($nome, $r->codice === 200 && $r->dati === ['v' => 1, 'authenticated' => false, 'user_id' => ANONYMOUS, 'username' => '', 'groups' => []]);
	}

	[$r] = chiama('127.0.0.1', 'abc', ['session_autologin' => 1, 'session_time' => time() - 86400 * 30] + $valida);
	verifica('ricordami, vecchia ma senza limite', $r->dati['authenticated'] === true);

	echo $falliti ? "$falliti falliti\n" : "ok\n";
	exit($falliti ? 1 : 0);
}
