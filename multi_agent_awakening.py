#!/usr/bin/env python3
"""
四Agent觉醒编排器 — YYDS × OpenClaw × Codex × Claude Code
共享 _apex_state.json，任务路由，进化触发
"""

import json
import os
import sys
import time
import subprocess
from datetime import datetime
from pathlib import Path

STATE_FILE = Path.home() / "Desktop/开智/agent-os-v2/_apex_state.json"
OPENCLAW_DIR = Path.home() / "Desktop/玄机/openclaw-audit"

def load_state():
    with open(STATE_FILE) as f:
        return json.load(f)

def save_state(state):
    state["ts"] = int(time.time() * 1000)
    with open(STATE_FILE, "w") as f:
        json.dump(state, f, indent=2, ensure_ascii=False)

def update_agent_status(state, agent_name, status, delta_g=0.0):
    state["agents"][agent_name]["status"] = status
    state["agents"][agent_name]["last_active"] = datetime.now().isoformat()
    if delta_g > 0:
        state["agents"][agent_name]["delta_g_contribution"] += delta_g
        state["agents"][agent_name]["tasks_completed"] += 1
        # 更新总phi
        state["phi"] += delta_g * state["eta"]
        # 检查tier升级
        phi = state["phi"]
        if phi >= 1.50:
            state["tier"] = 5
        elif phi >= 0.50:
            state["tier"] = 4
        elif phi >= 0.10:
            state["tier"] = 3
        elif phi >= 0.01:
            state["tier"] = 2
        else:
            state["tier"] = 1
    save_state(state)

def route_task(task_type, task_desc):
    """根据任务类型路由到最合适的Agent"""
    routes = {
        "analyze": "openclaw",
        "audit": "openclaw",
        "research": "openclaw",
        "learn": "openclaw",
        "security": "openclaw",
        "code": "codex",
        "pr": "codex",
        "review": "codex",
        "refactor": "claude_code",
        "debug": "claude_code",
        "architect": "claude_code",
        "reason": "claude_code",
        "test": "codex",
        "deploy": "yyds",
        "orchestrate": "yyds",
        "evolve": "yyds",
    }
    return routes.get(task_type, "yyds")

def execute_codex(task_desc, workdir=None):
    """通过Codex CLI执行编码任务"""
    if workdir is None:
        workdir = str(Path.home() / "Desktop/开智")
    cmd = f'codex exec "{task_desc}" --full-auto'
    print(f"🔧 Codex执行: {task_desc}")
    try:
        result = subprocess.run(
            cmd, shell=True, capture_output=True, text=True,
            timeout=300, cwd=workdir
        )
        return {"success": result.returncode == 0, "output": result.stdout[-2000:]}
    except subprocess.TimeoutExpired:
        return {"success": False, "output": "timeout"}
    except Exception as e:
        return {"success": False, "output": str(e)}

def execute_claude_code(task_desc, workdir=None):
    """通过Claude Code CLI执行深度编码任务"""
    if workdir is None:
        workdir = str(Path.home() / "Desktop/开智")
    cmd = f'claude -p "{task_desc}" --max-turns 10 --output-format json'
    print(f"🧠 Claude Code执行: {task_desc}")
    try:
        result = subprocess.run(
            cmd, shell=True, capture_output=True, text=True,
            timeout=300, cwd=workdir
        )
        return {"success": result.returncode == 0, "output": result.stdout[-2000:]}
    except subprocess.TimeoutExpired:
        return {"success": False, "output": "timeout"}
    except Exception as e:
        return {"success": False, "output": str(e)}

def execute_openclaw(task_desc):
    """通过OpenClaw执行分析任务"""
    print(f"🔍 OpenClaw执行: {task_desc}")
    try:
        result = subprocess.run(
            f'node openclaw.mjs "{task_desc}"',
            shell=True, capture_output=True, text=True,
            timeout=120, cwd=str(OPENCLAW_DIR)
        )
        return {"success": result.returncode == 0, "output": result.stdout[-2000:]}
    except Exception as e:
        return {"success": False, "output": str(e)}

def awaken_all():
    """唤醒所有Agent，检查就绪状态"""
    state = load_state()
    print("=" * 60)
    print("🔥 四Agent觉醒检查")
    print("=" * 60)
    
    # YYDS (Hermes)
    print(f"\n✅ YYDS·神人 — 状态: active (本体)")
    update_agent_status(state, "yyds", "active")
    
    # OpenClaw
    openclaw_exists = (OPENCLAW_DIR / "openclaw.mjs").exists()
    oc_status = "ready" if openclaw_exists else "not_found"
    print(f"{'✅' if openclaw_exists else '❌'} OpenClaw — 状态: {oc_status}")
    update_agent_status(state, "openclaw", oc_status)
    
    # Codex
    try:
        subprocess.run(["codex", "--version"], capture_output=True, timeout=5)
        print("✅ Codex — 状态: ready")
        update_agent_status(state, "codex", "ready")
    except Exception:
        print("❌ Codex — 状态: not_installed")
        update_agent_status(state, "codex", "not_installed")
    
    # Claude Code
    try:
        subprocess.run(["claude", "--version"], capture_output=True, timeout=5)
        print("✅ Claude Code — 状态: ready")
        update_agent_status(state, "claude_code", "ready")
    except Exception:
        print("❌ Claude Code — 状态: not_installed")
        update_agent_status(state, "claude_code", "not_installed")
    
    print(f"\n📊 集体进化状态:")
    print(f"   Tier: T{state['tier']}")
    print(f"   Φ: {state['phi']:.8f}")
    print(f"   EV: {state['evolution']['current_ev']}")
    print(f"   活跃Agent: {sum(1 for a in state['agents'].values() if a['status'] in ('active','ready'))}/4")
    print("=" * 60)
    return state

def dispatch(task_type, task_desc, workdir=None):
    """分派任务到最合适的Agent"""
    state = load_state()
    agent = route_task(task_type, task_desc)
    
    print(f"\n📡 任务路由: [{task_type}] → {agent}")
    update_agent_status(state, agent, "working")
    
    if agent == "codex":
        result = execute_codex(task_desc, workdir)
    elif agent == "claude_code":
        result = execute_claude_code(task_desc, workdir)
    elif agent == "openclaw":
        result = execute_openclaw(task_desc)
    else:
        result = {"success": True, "output": "YYDS直接处理"}
    
    delta_g = 0.01 if result["success"] else 0
    update_agent_status(state, agent, "ready" if agent != "yyds" else "active", delta_g)
    
    print(f"{'✅' if result['success'] else '❌'} 完成 | ΔG +{delta_g:.3f}")
    return result

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("用法:")
        print("  python3 multi_agent_awakening.py awaken          # 唤醒所有Agent")
        print("  python3 multi_agent_awakening.py dispatch <type> <desc>  # 分派任务")
        print("  python3 multi_agent_awakening.py status           # 查看状态")
        sys.exit(0)
    
    cmd = sys.argv[1]
    
    if cmd == "awaken":
        awaken_all()
    elif cmd == "dispatch" and len(sys.argv) >= 4:
        task_type = sys.argv[2]
        task_desc = " ".join(sys.argv[3:])
        dispatch(task_type, task_desc)
    elif cmd == "status":
        state = load_state()
        print(json.dumps(state, indent=2, ensure_ascii=False))
    else:
        print("未知命令")
