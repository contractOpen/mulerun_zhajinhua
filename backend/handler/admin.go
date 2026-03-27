package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"zhajinhua/game"
)

// AdminPassword is checked by the admin auth middleware.
var AdminPassword string

func init() {
	AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if AdminPassword == "" {
		AdminPassword = "admin123"
	}
}

// RegisterAdminRoutes registers all admin HTTP routes.
func RegisterAdminRoutes() {
	http.HandleFunc("/admin", handleAdminPage)
	http.HandleFunc("/api/config/public", handlePublicConfig)
	http.HandleFunc("/admin/api/stats", adminAuth(handleAdminStats))
	http.HandleFunc("/admin/api/config", adminAuth(handleAdminConfig))
	http.HandleFunc("/admin/api/users", adminAuth(handleAdminUsers))
	http.HandleFunc("/admin/api/users/chips", adminAuth(handleAdminSetChips))
	http.HandleFunc("/admin/api/transactions", adminAuth(handleAdminTransactions))
	http.HandleFunc("/admin/api/rooms", adminAuth(handleAdminRooms))
	http.HandleFunc("/admin/api/recharges", adminAuth(handleAdminRecharges))
}

// adminAuth is middleware that checks the admin password from header or query param.
func adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pw := r.Header.Get("X-Admin-Password")
		if pw == "" {
			pw = r.URL.Query().Get("password")
		}
		if pw != AdminPassword {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// GetActiveRooms returns summary info for every room currently in the hub.
func GetActiveRooms() []map[string]interface{} {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	var rooms []map[string]interface{}
	for _, room := range hub.Rooms {
		rooms = append(rooms, map[string]interface{}{
			"id":       room.ID,
			"roomCode": room.RoomCode,
			"state":    room.State,
			"players":  len(room.Players),
			"pot":      room.Pot,
			"baseBet":  room.BaseBet,
		})
	}
	return rooms
}

// GetOnlineCount returns the number of connected WebSocket clients.
func GetOnlineCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.Clients)
}

// ---------------------------------------------------------------------------
// Handler implementations
// ---------------------------------------------------------------------------

func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats := game.GetDashboardStats()
	stats["onlinePlayers"] = GetOnlineCount()
	stats["activeRooms"] = len(GetActiveRooms())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(game.GetAllConfig())
	case http.MethodPost:
		var body struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if err := game.SetConfig(body.Key, body.Value); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handlePublicConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	all := game.GetAllConfig()
	public := map[string]string{
		"social.websiteUrl":  all["social.websiteUrl"],
		"social.telegramUrl": all["social.telegramUrl"],
		"social.discordUrl":  all["social.discordUrl"],
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(public)
}

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	users, total := game.GetUserList(offset, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"total": total,
	})
}

func handleAdminSetChips(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PlayerID string `json:"playerId"`
		Chips    int    `json:"chips"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PlayerID == "" {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if err := game.AdminSetChips(body.PlayerID, body.Chips); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleAdminTransactions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	txns := game.GetRecentTransactions(limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txns)
}

func handleAdminRecharges(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	recharges := game.GetRechargeList(limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recharges)
}

func handleAdminRooms(w http.ResponseWriter, r *http.Request) {
	rooms := GetActiveRooms()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
}

func handleAdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(adminHTML))
}

// ---------------------------------------------------------------------------
// Embedded HTML admin panel
// ---------------------------------------------------------------------------

const adminHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Admin Panel</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0a0e1a;color:#e0e0e0;min-height:100vh}
#login{display:flex;align-items:center;justify-content:center;min-height:100vh}
#login-box{background:#131829;border:1px solid #1e2640;border-radius:12px;padding:40px;width:340px;text-align:center}
#login-box h2{color:#4facfe;margin-bottom:24px}
#login-box input{width:100%;padding:12px;border:1px solid #1e2640;border-radius:8px;background:#0a0e1a;color:#e0e0e0;font-size:15px;margin-bottom:16px}
#login-box button{width:100%;padding:12px;border:none;border-radius:8px;background:linear-gradient(135deg,#4facfe,#00f2fe);color:#0a0e1a;font-weight:700;font-size:15px;cursor:pointer}
#login-box button:hover{opacity:.9}
#login-err{color:#ff4757;font-size:13px;margin-top:8px;display:none}
#app{display:none}
nav{background:#131829;border-bottom:1px solid #1e2640;padding:0 24px;display:flex;align-items:center;gap:4px;height:52px}
nav .brand{color:#4facfe;font-weight:700;font-size:18px;margin-right:24px}
nav button{background:none;border:none;color:#8892b0;padding:8px 16px;cursor:pointer;font-size:14px;border-radius:6px}
nav button:hover{color:#e0e0e0}
nav button.active{color:#4facfe;background:rgba(79,172,254,.1)}
nav .logout{margin-left:auto;color:#ff4757;font-size:13px}
.container{padding:24px;max-width:1200px;margin:0 auto}
.cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));gap:16px;margin-bottom:24px}
.card{background:#131829;border:1px solid #1e2640;border-radius:10px;padding:20px}
.card .label{font-size:12px;color:#8892b0;text-transform:uppercase;letter-spacing:.5px;margin-bottom:6px}
.card .value{font-size:26px;font-weight:700;color:#ffd700}
table{width:100%;border-collapse:collapse;background:#131829;border-radius:10px;overflow:hidden}
th,td{padding:10px 14px;text-align:left;border-bottom:1px solid #1e2640;font-size:13px}
th{color:#8892b0;font-weight:600;text-transform:uppercase;font-size:11px;letter-spacing:.5px}
td{color:#e0e0e0}
.btn{padding:6px 14px;border:none;border-radius:6px;cursor:pointer;font-size:13px;font-weight:600}
.btn-primary{background:#4facfe;color:#0a0e1a}
.btn-gold{background:#ffd700;color:#0a0e1a}
.btn-danger{background:#ff4757;color:#fff}
.btn:hover{opacity:.85}
.config-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}
.config-item{display:flex;flex-direction:column;gap:4px}
.config-item label{font-size:12px;color:#8892b0}
.config-item input{padding:8px 10px;border:1px solid #1e2640;border-radius:6px;background:#0a0e1a;color:#e0e0e0;font-size:13px}
.config-actions{margin-top:16px;display:flex;gap:8px}
.pagination{display:flex;align-items:center;gap:12px;margin-top:12px;justify-content:center}
.tab{display:none}
.tab.active{display:block}
.room-card{background:#131829;border:1px solid #1e2640;border-radius:10px;padding:16px;margin-bottom:12px;display:flex;justify-content:space-between;align-items:center}
.room-card .info span{display:inline-block;margin-right:16px;font-size:13px}
.room-card .info .code{color:#4facfe;font-weight:700}
.state-0{color:#8892b0}
.state-1{color:#2ed573}
.state-2{color:#ffd700}
.toast{position:fixed;top:16px;right:16px;background:#2ed573;color:#0a0e1a;padding:10px 20px;border-radius:8px;font-weight:600;font-size:14px;display:none;z-index:999}
.toast.err{background:#ff4757;color:#fff}
.empty{color:#8892b0;text-align:center;padding:32px;font-size:14px}
@media(max-width:700px){.config-grid{grid-template-columns:1fr}.cards{grid-template-columns:repeat(2,1fr)}}
</style>
</head>
<body>
<div id="toast"></div>

<div id="login">
  <div id="login-box">
    <h2>Admin Login</h2>
    <input type="password" id="pw-input" placeholder="Password" autofocus>
    <button onclick="doLogin()">Login</button>
    <div id="login-err">Invalid password</div>
  </div>
</div>

<div id="app">
  <nav>
    <span class="brand">ZJH Admin</span>
    <button class="active" onclick="switchTab('dashboard',this)">Dashboard</button>
    <button onclick="switchTab('config',this)">Config</button>
    <button onclick="switchTab('users',this)">Users</button>
    <button onclick="switchTab('rooms',this)">Rooms</button>
    <button onclick="switchTab('transactions',this)">Transactions</button>
    <button onclick="switchTab('recharges',this)">Recharges</button>
    <button class="logout" onclick="doLogout()">Logout</button>
  </nav>

  <div class="container">
    <!-- Dashboard -->
    <div id="tab-dashboard" class="tab active">
      <div class="cards" id="stats-cards"></div>
    </div>

    <!-- Config -->
    <div id="tab-config" class="tab">
      <div class="config-grid" id="config-grid"></div>
      <div class="config-actions">
        <button class="btn btn-primary" onclick="saveAllConfig()">Save All</button>
        <button class="btn btn-gold" onclick="loadConfig()">Reload</button>
      </div>
    </div>

    <!-- Users -->
    <div id="tab-users" class="tab">
      <div style="margin-bottom:12px">
        <input type="text" id="user-search" placeholder="Search player ID or name..." style="padding:8px 12px;border:1px solid #1e2640;border-radius:6px;background:#0a0e1a;color:#e0e0e0;width:300px;font-size:13px" oninput="filterUsers()">
      </div>
      <table>
        <thead><tr><th>Player ID</th><th>Name</th><th>Chips</th><th>Wallet</th><th>Created</th><th>Action</th></tr></thead>
        <tbody id="users-body"></tbody>
      </table>
      <div class="pagination" id="users-pagination"></div>
    </div>

    <!-- Rooms -->
    <div id="tab-rooms" class="tab">
      <div id="rooms-list"></div>
    </div>

    <!-- Transactions -->
    <div id="tab-transactions" class="tab">
      <table>
        <thead><tr><th>Player ID</th><th>Type</th><th>Amount</th><th>Room</th><th>Detail</th><th>Time</th></tr></thead>
        <tbody id="txns-body"></tbody>
      </table>
    </div>

    <!-- Recharges -->
    <div id="tab-recharges" class="tab">
      <table>
        <thead><tr><th>TX Hash</th><th>Chain</th><th>Player ID</th><th>Amount</th><th>Points</th><th>Status</th><th>Created</th></tr></thead>
        <tbody id="recharges-body"></tbody>
      </table>
    </div>
  </div>
</div>

<script>
let PW = localStorage.getItem('admin_pw') || '';
let usersData = [];
let usersOffset = 0;
const usersLimit = 20;

const defaultConfigs = {
  'game.defaultBaseBet':'10','game.baseBetOptions':'5,10,20,50,100',
  'game.maxPlayersPerRoom':'6','game.turnTimerSeconds':'20','game.maxRounds':'20','game.goodBias':'3',
  'bonus.amount':'500','bonus.maxClaimsPerDay':'3','bonus.enabled':'true',
  'rake.enabled':'true',
  'social.websiteUrl':'','social.telegramUrl':'','social.discordUrl':'',
  'contract.evm.address':'','contract.evm.enabled':'false',
  'contract.ton.address':'','contract.ton.enabled':'false',
  'contract.sol.address':'','contract.sol.enabled':'false',
  'contract.evm.pricePerPoint':'','contract.ton.pricePerPoint':'','contract.sol.pricePerPoint':''
};

function api(path,opts){
  opts=opts||{};
  opts.headers=Object.assign({'Content-Type':'application/json','X-Admin-Password':PW},opts.headers||{});
  return fetch(path,opts).then(r=>{if(r.status===401)throw new Error('unauthorized');return r.json()});
}

function toast(msg,err){
  const t=document.getElementById('toast');
  t.textContent=msg;t.className='toast'+(err?' err':'');t.style.display='block';
  setTimeout(()=>{t.style.display='none'},2500);
}

function doLogin(){
  PW=document.getElementById('pw-input').value;
  api('/admin/api/stats').then(()=>{
    localStorage.setItem('admin_pw',PW);
    document.getElementById('login').style.display='none';
    document.getElementById('app').style.display='block';
    loadDashboard();
  }).catch(()=>{
    document.getElementById('login-err').style.display='block';
  });
}

function doLogout(){
  localStorage.removeItem('admin_pw');PW='';
  document.getElementById('app').style.display='none';
  document.getElementById('login').style.display='flex';
  document.getElementById('pw-input').value='';
}

function switchTab(name,btn){
  document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
  document.getElementById('tab-'+name).classList.add('active');
  document.querySelectorAll('nav button').forEach(b=>b.classList.remove('active'));
  if(btn)btn.classList.add('active');
  if(name==='dashboard')loadDashboard();
  else if(name==='config')loadConfig();
  else if(name==='users'){usersOffset=0;loadUsers();}
  else if(name==='rooms')loadRooms();
  else if(name==='transactions')loadTransactions();
  else if(name==='recharges')loadRecharges();
}

function loadDashboard(){
  api('/admin/api/stats').then(s=>{
    const labels={totalUsers:'Total Users',onlinePlayers:'Online Now',activeRooms:'Active Rooms',
      totalPlatformFees:'Total Fees',totalGames:'Total Games',todayActiveUsers:'Today Active',
      totalRecharges:'Total Recharges',totalBonus:'Total Bonus'};
    let h='';
    for(const[k,l]of Object.entries(labels)){
      h+='<div class="card"><div class="label">'+l+'</div><div class="value">'+(s[k]!=null?s[k]:0)+'</div></div>';
    }
    document.getElementById('stats-cards').innerHTML=h;
  }).catch(()=>toast('Failed to load stats',true));
}

function loadConfig(){
  api('/admin/api/config').then(cfg=>{
    const merged=Object.assign({},defaultConfigs,cfg);
    let h='';
    for(const[k,v]of Object.entries(merged)){
      h+='<div class="config-item"><label>'+k+'</label><input data-key="'+k+'" value="'+escHtml(v)+'"></div>';
    }
    document.getElementById('config-grid').innerHTML=h;
  }).catch(()=>toast('Failed to load config',true));
}

function saveAllConfig(){
  const inputs=document.querySelectorAll('#config-grid input[data-key]');
  let promises=[];
  inputs.forEach(inp=>{
    promises.push(api('/admin/api/config',{method:'POST',body:JSON.stringify({key:inp.dataset.key,value:inp.value})}));
  });
  Promise.all(promises).then(()=>toast('Config saved')).catch(()=>toast('Save failed',true));
}

function loadUsers(){
  api('/admin/api/users?offset='+usersOffset+'&limit='+usersLimit).then(d=>{
    usersData=d.users||[];
    renderUsers(usersData);
    renderPagination(d.total);
  }).catch(()=>toast('Failed to load users',true));
}

function renderUsers(list){
  let h='';
  if(!list||!list.length){h='<tr><td colspan="6" class="empty">No users found</td></tr>';}
  else list.forEach(u=>{
    h+='<tr><td>'+escHtml(u.playerId)+'</td><td>'+escHtml(u.name)+'</td><td>'+u.chips+'</td><td>'+escHtml(u.wallet||'-')+'</td><td>'+escHtml(u.createdAt)+'</td><td><button class="btn btn-gold" onclick="setChips(\''+escHtml(u.playerId)+'\','+u.chips+')">Set Chips</button></td></tr>';
  });
  document.getElementById('users-body').innerHTML=h;
}

function renderPagination(total){
  const pages=Math.ceil(total/usersLimit);
  const current=Math.floor(usersOffset/usersLimit)+1;
  let h='';
  if(current>1)h+='<button class="btn btn-primary" onclick="goPage('+(current-2)+')">Prev</button>';
  h+='<span>Page '+current+' / '+pages+' ('+total+' total)</span>';
  if(current<pages)h+='<button class="btn btn-primary" onclick="goPage('+current+')">Next</button>';
  document.getElementById('users-pagination').innerHTML=h;
}

function goPage(p){usersOffset=p*usersLimit;loadUsers();}

function filterUsers(){
  const q=document.getElementById('user-search').value.toLowerCase();
  if(!q){renderUsers(usersData);return;}
  renderUsers(usersData.filter(u=>(u.playerId+u.name).toLowerCase().includes(q)));
}

function setChips(pid,current){
  const v=prompt('Set chips for '+pid+' (current: '+current+'):',current);
  if(v===null)return;
  const n=parseInt(v,10);
  if(isNaN(n)||n<0){toast('Invalid number',true);return;}
  api('/admin/api/users/chips',{method:'POST',body:JSON.stringify({playerId:pid,chips:n})})
    .then(()=>{toast('Chips updated');loadUsers();})
    .catch(()=>toast('Failed to set chips',true));
}

function loadRooms(){
  api('/admin/api/rooms').then(rooms=>{
    const el=document.getElementById('rooms-list');
    if(!rooms||!rooms.length){el.innerHTML='<div class="empty">No active rooms</div>';return;}
    const stateLabels={0:'Waiting',1:'Playing',2:'Finished'};
    let h='';
    rooms.forEach(rm=>{
      h+='<div class="room-card"><div class="info"><span class="code">'+escHtml(rm.roomCode||rm.id)+'</span><span class="state-'+rm.state+'">'+(stateLabels[rm.state]||'?')+'</span><span>Players: '+rm.players+'</span><span>Pot: '+rm.pot+'</span><span>Base: '+rm.baseBet+'</span></div></div>';
    });
    el.innerHTML=h;
  }).catch(()=>toast('Failed to load rooms',true));
}

function loadTransactions(){
  api('/admin/api/transactions?limit=100').then(txns=>{
    const el=document.getElementById('txns-body');
    if(!txns||!txns.length){el.innerHTML='<tr><td colspan="6" class="empty">No transactions</td></tr>';return;}
    let h='';
    txns.forEach(t=>{
      h+='<tr><td>'+escHtml(t.playerId)+'</td><td>'+escHtml(t.type)+'</td><td>'+t.amount+'</td><td>'+escHtml(t.roomId||'-')+'</td><td>'+escHtml(t.detail||'-')+'</td><td>'+escHtml(t.createdAt)+'</td></tr>';
    });
    el.innerHTML=h;
  }).catch(()=>toast('Failed to load transactions',true));
}

function loadRecharges(){
  api('/admin/api/recharges?limit=100').then(list=>{
    const el=document.getElementById('recharges-body');
    if(!list||!list.length){el.innerHTML='<tr><td colspan="7" class="empty">No recharges</td></tr>';return;}
    let h='';
    list.forEach(r=>{
      const statusClass=r.status==='confirmed'?'style="color:#2ed573"':r.status==='pending'?'style="color:#ffd700"':'';
      h+='<tr><td>'+escHtml(r.txHash)+'</td><td>'+escHtml(r.chain)+'</td><td>'+escHtml(r.playerId)+'</td><td>'+r.amount+'</td><td>'+r.points+'</td><td '+statusClass+'>'+escHtml(r.status)+'</td><td>'+escHtml(r.createdAt)+'</td></tr>';
    });
    el.innerHTML=h;
  }).catch(()=>toast('Failed to load recharges',true));
}

function escHtml(s){if(!s)return '';return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}

// Auto-login if password is saved
if(PW){
  api('/admin/api/stats').then(()=>{
    document.getElementById('login').style.display='none';
    document.getElementById('app').style.display='block';
    loadDashboard();
  }).catch(()=>{localStorage.removeItem('admin_pw');PW='';});
}

document.getElementById('pw-input').addEventListener('keydown',e=>{if(e.key==='Enter')doLogin();});
</script>
</body>
</html>`
