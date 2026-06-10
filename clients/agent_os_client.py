#!/usr/bin/env python3
"""
Agent OS v2 Python Client
四Agent通过此客户端与Agent OS v2交互：
  - 提交任务到nanobot调度器
  - 查询任务结果
  - 查看系统状态和进化数据
  - 任务结果自动上链存证 + MOSS-AGI进化量化

用法:
    from agent_os_client import AgentOS
    os = AgentOS("http://localhost:8200")
    os.submit_task("分析代码", payload="review src/auth.py")
    status = os.status()
    evolution = os.evolution()
"""
import json
import time
import urllib.request
import urllib.error
from typing import Optional, Dict, Any, List


class AgentOS:
    """Agent OS v2 客户端"""

    def __init__(self, base_url: str = "http://localhost:8200", agent_id: str = "yyds"):
        self.base_url = base_url.rstrip("/")
        self.agent_id = agent_id

    def _request(self, method: str, path: str, data: dict = None) -> dict:
        url = f"{self.base_url}{path}"
        body = json.dumps(data).encode() if data else None
        req = urllib.request.Request(url, data=body, method=method)
        req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                return json.loads(resp.read())
        except urllib.error.HTTPError as e:
            return {"error": e.read().decode(), "status": e.code}
        except urllib.error.URLError:
            return {"error": "Agent OS v2 not reachable", "status": 0}

    # ═══════════════════════════════════════════════
    # 任务API
    # ═══════════════════════════════════════════════

    def submit_task(self, name: str, payload: str = "", priority: int = 1, timeout: int = 30) -> dict:
        """提交任务到nanobot调度器，自动分配到最优Agent节点"""
        return self._request("POST", "/api/task/submit", {
            "agent_id": self.agent_id,
            "task_name": name,
            "priority": priority,
            "payload": payload,
            "timeout_seconds": timeout,
        })

    def task_result(self, task_id: str) -> dict:
        """查询任务结果"""
        return self._request("GET", f"/api/task/result?task_id={task_id}")

    def task_callback(self, task_id: str, success: bool, data: str = "", error: str = "") -> dict:
        """任务完成回调 — 结果上链 + MOSS-AGI进化"""
        return self._request("POST", "/api/task/callback", {
            "task_id": task_id,
            "agent_id": self.agent_id,
            "success": success,
            "data": data,
            "error": error,
            "duration_ms": 0,
        })

    # ═══════════════════════════════════════════════
    # Agent API
    # ═══════════════════════════════════════════════

    def register(self, address: str = "localhost:8110", capacity: int = 5, labels: dict = None) -> dict:
        """注册当前Agent为nanobot节点"""
        return self._request("POST", "/api/agent/register", {
            "agent_id": self.agent_id,
            "address": address,
            "capacity": capacity,
            "labels": labels or {},
        })

    def agent_stats(self) -> list:
        """获取所有Agent统计"""
        return self._request("GET", "/api/agent/stats")

    # ═══════════════════════════════════════════════
    # 系统状态API
    # ═══════════════════════════════════════════════

    def status(self) -> dict:
        """系统总览：进化状态 + 调度器状态 + 区块链状态"""
        return self._request("GET", "/api/status")

    def evolution(self) -> dict:
        """MOSS-AGI进化详情：EV/ΔG/tier/基因池"""
        return self._request("GET", "/api/evolution")

    def blockchain(self) -> dict:
        """区块链状态：区块列表 + 待处理交易 + 跨链桥"""
        return self._request("GET", "/api/blockchain")

    def health(self) -> dict:
        """健康检查"""
        return self._request("GET", "/api/health")

    # ═══════════════════════════════════════════════
    # 便捷方法
    # ═══════════════════════════════════════════════

    def is_alive(self) -> bool:
        """Agent OS v2是否在运行"""
        r = self.health()
        return r.get("status") == "ok"

    def ev(self) -> float:
        """当前EV值"""
        return self.evolution().get("ev", 0)

    def tier(self) -> int:
        """当前Tier"""
        return self.evolution().get("tier", 0)

    def submit_and_wait(self, name: str, payload: str = "", timeout: int = 30, poll_interval: float = 1.0) -> dict:
        """提交任务并等待结果"""
        resp = self.submit_task(name, payload, timeout=timeout)
        task_id = resp.get("task_id")
        if not task_id:
            return resp
        deadline = time.time() + timeout
        while time.time() < deadline:
            result = self.task_result(task_id)
            if "error" not in result:
                return result
            time.sleep(poll_interval)
        return {"error": "timeout", "task_id": task_id}


# ═══════════════════════════════════════════════
# CLI用法
# ═══════════════════════════════════════════════
if __name__ == "__main__":
    import sys

    agent_id = sys.argv[1] if len(sys.argv) > 1 else "yyds"
    os_client = AgentOS(agent_id=agent_id)

    print(f"Agent OS v2 Client (agent={agent_id})")
    print(f"{'='*50}")

    if not os_client.is_alive():
        print("❌ Agent OS v2 not running on :8200")
        sys.exit(1)

    print("✅ Connected to Agent OS v2")

    status = os_client.status()
    evo = status.get("evolution", {})
    sched = status.get("scheduler", {})
    print(f"  EV={evo.get('ev', 0):.4f} tier=T{evo.get('tier', 0)}")
    print(f"  Nodes={sched.get('nodes', 0)} tasks={sched.get('completed', 0)}")
    print(f"  Blockchain: {status.get('blockchain', {}).get('blocks', 0)} blocks")
