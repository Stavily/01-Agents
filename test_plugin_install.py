#!/usr/bin/env python3
"""
Test script to simulate plugin installation instruction
"""
import json
import requests
import time

# Test instruction payload as provided by the user
test_instruction = {
    "instruction": {
        "id": "0aeb8570-9b0e-4dd5-b567-76261e29f0d7",
        "agent_id": "65bb7a51-1812-4a37-90d5-3e8a859ba972",
        "plugin_id": "e56434ca-9756-446e-92cd-f7545cb5b7b2",
        "created_by": None,
        "status": "pending",
        "priority": "high",
        "instruction_type": "plugin_install",
        "source": "web-ui",
        "plugin_configuration": {
            "entrypoint": "main.py",
            "plugin_url": "https://github.com/stavily/06-plugins"
        },
        "timeout_seconds": 300,
        "max_retries": 3,
        "retry_count": 0,
        "execution_log": []
    },
    "status": "instruction_delivered",
    "next_poll_interval": 5
}

def test_plugin_installation():
    """Test plugin installation by simulating the instruction"""
    print("Testing plugin installation...")
    print("Instruction payload:")
    print(json.dumps(test_instruction, indent=2))
    
    # Check if the plugin directory exists
    import os
    plugin_dir = "/home/eduardez/Workspace/Stavily/01-Agents/test-agents/sensor-agent-01/config/plugins"
    print(f"\nPlugin directory: {plugin_dir}")
    print(f"Directory exists: {os.path.exists(plugin_dir)}")
    
    if os.path.exists(plugin_dir):
        print("Contents before installation:")
        try:
            contents = os.listdir(plugin_dir)
            print(f"  {contents}")
        except Exception as e:
            print(f"  Error listing contents: {e}")
    
    # Note: This would normally be sent to the agent's polling endpoint
    # For testing purposes, we'll just verify the instruction format
    print("\nInstruction validation:")
    instruction = test_instruction["instruction"]
    
    # Check required fields for plugin_install
    required_fields = ["id", "plugin_id", "plugin_configuration"]
    for field in required_fields:
        if field in instruction:
            print(f"  ✓ {field}: {instruction[field]}")
        else:
            print(f"  ✗ {field}: missing")
    
    # Check plugin_configuration
    plugin_config = instruction.get("plugin_configuration", {})
    if "plugin_url" in plugin_config:
        print(f"  ✓ plugin_url: {plugin_config['plugin_url']}")
    else:
        print("  ✗ plugin_url: missing")
    
    print("\nTest completed. The agent should process this instruction and:")
    print("1. Download the plugin from the GitHub URL")
    print("2. Install it to the plugins directory")
    print("3. Update the execution_log with download progress")
    print("4. Return a success/failure result")

if __name__ == "__main__":
    test_plugin_installation() 