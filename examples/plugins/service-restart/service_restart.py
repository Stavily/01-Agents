#!/usr/bin/env python3
"""
Service Restart Action Plugin for Stavily Action Agents

This plugin restarts system services using systemctl or other service managers.
"""

import json
import os
import sys
import subprocess
import time
import logging
import re
from datetime import datetime
from typing import Dict, Any, Optional, List

class ServiceRestartPlugin:
    """Service restart action plugin implementation."""
    
    def __init__(self):
        self.config = {}
        self.running = False
        self.status = "stopped"
        self.start_time = None
        self.logger = self._setup_logging()
        self.allowed_services = []
        self.service_manager = "systemctl"
    
    def _setup_logging(self) -> logging.Logger:
        """Setup plugin logging."""
        logger = logging.getLogger('service_restart')
        logger.setLevel(logging.INFO)
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
        return logger
    
    def get_info(self) -> Dict[str, Any]:
        """Return plugin metadata."""
        return {
            "id": "service-restart",
            "name": "Service Restart",
            "description": "Restarts system services using systemctl or other service managers",
            "version": "1.0.0",
            "author": "Stavily Team",
            "license": "MIT",
            "type": "action",
            "tags": ["system", "service", "restart"],
            "categories": ["system-management"],
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat()
        }
    
    def initialize(self, config: Dict[str, Any]) -> bool:
        """Initialize the plugin with configuration."""
        try:
            self.config = config
            self.allowed_services = config.get("allowed_services", [])
            self.service_manager = config.get("service_manager", "systemctl")
            
            # Validate service manager
            if self.service_manager not in ["systemctl", "service", "docker"]:
                raise ValueError(f"Unsupported service manager: {self.service_manager}")
            
            # Check if service manager is available
            if not self._check_service_manager():
                raise RuntimeError(f"Service manager '{self.service_manager}' not available")
                
            self.status = "initialized"
            self.logger.info(f"Service Restart initialized with manager={self.service_manager}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to initialize plugin: {e}")
            self.status = "error"
            return False
    
    def start(self) -> bool:
        """Start the plugin execution."""
        try:
            if self.running:
                self.logger.warning("Plugin is already running")
                return True
                
            self.running = True
            self.status = "running"
            self.start_time = datetime.now()
            self.logger.info("Service Restart plugin started")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to start plugin: {e}")
            self.status = "error"
            return False
    
    def stop(self) -> bool:
        """Stop the plugin execution."""
        try:
            self.running = False
            self.status = "stopped"
            self.logger.info("Service Restart plugin stopped")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to stop plugin: {e}")
            return False
    
    def get_status(self) -> str:
        """Return the current plugin status."""
        return self.status
    
    def get_health(self) -> Dict[str, Any]:
        """Return plugin health information."""
        health = {
            "status": "healthy" if self.running else "unhealthy",
            "message": "Plugin is running normally" if self.running else "Plugin is not running",
            "last_check": datetime.now().isoformat(),
            "uptime": 0,
            "error_count": 0,
            "metrics": {
                "service_manager": self.service_manager,
                "allowed_services_count": len(self.allowed_services)
            }
        }
        
        if self.start_time:
            uptime = (datetime.now() - self.start_time).total_seconds()
            health["uptime"] = uptime
            
        return health
    
    def execute_action(self, action_request: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a service restart action."""
        if not self.running:
            return self._create_error_result(action_request["id"], "Plugin is not running")
        
        start_time = datetime.now()
        action_id = action_request["id"]
        parameters = action_request.get("parameters", {})
        
        result = {
            "id": action_id,
            "status": "running",
            "started_at": start_time.isoformat(),
            "metadata": {
                "plugin_id": "service-restart",
                "plugin_version": "1.0.0",
                "execution_host": os.uname().nodename
            }
        }
        
        try:
            # Extract and validate service name
            service_name = parameters.get("service_name")
            if not service_name:
                return self._create_error_result(action_id, "service_name parameter is required", start_time)
            
            if not self._is_valid_service_name(service_name):
                return self._create_error_result(action_id, f"Invalid service name: {service_name}", start_time)
            
            # Check if service is allowed
            if self.allowed_services and service_name not in self.allowed_services:
                return self._create_error_result(action_id, f"Service '{service_name}' is not in allowed list", start_time)
            
            # Get restart method
            method = parameters.get("method", self.service_manager)
            if method not in ["systemctl", "service", "docker"]:
                return self._create_error_result(action_id, f"Unsupported method: {method}", start_time)
            
            # Execute restart command
            command_result = self._execute_restart_command(service_name, method)
            
            if command_result["success"]:
                result.update({
                    "status": "completed",
                    "data": {
                        "service_name": service_name,
                        "method": method,
                        "command": command_result["command"],
                        "output": command_result["output"],
                        "success": True
                    }
                })
                self.logger.info(f"Successfully restarted service: {service_name}")
            else:
                result.update({
                    "status": "failed",
                    "error": command_result["error"],
                    "data": {
                        "service_name": service_name,
                        "method": method,
                        "command": command_result["command"],
                        "output": command_result["output"]
                    }
                })
                self.logger.error(f"Failed to restart service {service_name}: {command_result['error']}")
            
        except Exception as e:
            result.update({
                "status": "failed",
                "error": str(e)
            })
            self.logger.error(f"Action execution error: {e}")
        
        # Finalize result
        result.update({
            "completed_at": datetime.now().isoformat(),
            "duration": (datetime.now() - start_time).total_seconds()
        })
        
        return result
    
    def get_action_config(self) -> Dict[str, Any]:
        """Return the action configuration schema."""
        return {
            "schema": {
                "service_name": {
                    "type": "string",
                    "description": "Name of the service to restart",
                    "required": True,
                    "min_length": 1,
                    "max_length": 100,
                    "examples": ["nginx", "apache2", "mysql", "postgresql"]
                },
                "method": {
                    "type": "string",
                    "description": "Method to use for restarting the service",
                    "default": "systemctl",
                    "required": False,
                    "enum": ["systemctl", "service", "docker"],
                    "examples": ["systemctl", "service", "docker"]
                },
                "wait_for_status": {
                    "type": "boolean",
                    "description": "Wait for service to be active after restart",
                    "default": True,
                    "required": False
                },
                "timeout": {
                    "type": "integer",
                    "description": "Timeout in seconds for the restart operation",
                    "default": 30,
                    "minimum": 5,
                    "maximum": 300
                }
            },
            "required": ["service_name"],
            "examples": [
                {
                    "service_name": "nginx",
                    "method": "systemctl",
                    "wait_for_status": True
                },
                {
                    "service_name": "web-app",
                    "method": "docker",
                    "timeout": 60
                }
            ],
            "description": "Service restart configuration",
            "timeout": 300
        }
    
    def _create_error_result(self, action_id: str, error: str, start_time: Optional[datetime] = None) -> Dict[str, Any]:
        """Create an error result."""
        if start_time is None:
            start_time = datetime.now()
            
        return {
            "id": action_id,
            "status": "failed",
            "error": error,
            "started_at": start_time.isoformat(),
            "completed_at": datetime.now().isoformat(),
            "duration": (datetime.now() - start_time).total_seconds(),
            "metadata": {
                "plugin_id": "service-restart",
                "plugin_version": "1.0.0"
            }
        }
    
    def _is_valid_service_name(self, name: str) -> bool:
        """Validate service name format."""
        # Basic validation: alphanumeric, hyphens, underscores, dots
        pattern = r'^[a-zA-Z0-9._-]+$'
        return bool(re.match(pattern, name)) and len(name) <= 100
    
    def _check_service_manager(self) -> bool:
        """Check if the service manager is available."""
        try:
            if self.service_manager == "systemctl":
                subprocess.run(["systemctl", "--version"], 
                             capture_output=True, check=True, timeout=5)
            elif self.service_manager == "service":
                subprocess.run(["service", "--version"], 
                             capture_output=True, timeout=5)
            elif self.service_manager == "docker":
                subprocess.run(["docker", "--version"], 
                             capture_output=True, check=True, timeout=5)
            return True
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired, FileNotFoundError):
            return False
    
    def _execute_restart_command(self, service_name: str, method: str) -> Dict[str, Any]:
        """Execute the restart command."""
        try:
            # Build command based on method
            if method == "systemctl":
                cmd = ["systemctl", "restart", service_name]
            elif method == "service":
                cmd = ["service", service_name, "restart"]
            elif method == "docker":
                cmd = ["docker", "restart", service_name]
            else:
                return {
                    "success": False,
                    "error": f"Unsupported method: {method}",
                    "command": "",
                    "output": ""
                }
            
            # For demonstration purposes, we'll simulate the command
            # In production, you would actually execute it with proper permissions
            if os.getenv("STAVILY_DEMO_MODE", "true").lower() == "true":
                return self._simulate_restart_command(cmd, service_name)
            else:
                # Real execution (requires proper permissions)
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=30
                )
                
                return {
                    "success": result.returncode == 0,
                    "error": result.stderr if result.returncode != 0 else "",
                    "command": " ".join(cmd),
                    "output": result.stdout
                }
                
        except subprocess.TimeoutExpired:
            return {
                "success": False,
                "error": "Command timed out",
                "command": " ".join(cmd),
                "output": ""
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
                "command": " ".join(cmd) if 'cmd' in locals() else "",
                "output": ""
            }
    
    def _simulate_restart_command(self, cmd: List[str], service_name: str) -> Dict[str, Any]:
        """Simulate restart command for demonstration."""
        # Simulate some processing time
        time.sleep(1)
        
        # Simulate success for most services
        if service_name in ["nginx", "apache2", "mysql", "postgresql", "redis", "docker"]:
            return {
                "success": True,
                "error": "",
                "command": " ".join(cmd),
                "output": f"Service {service_name} restarted successfully (simulated)"
            }
        else:
            return {
                "success": False,
                "error": f"Service {service_name} not found (simulated)",
                "command": " ".join(cmd),
                "output": ""
            }


def main():
    """Main plugin entry point."""
    plugin = ServiceRestartPlugin()
    
    # Plugin communication protocol
    while True:
        try:
            # Read command from stdin
            line = sys.stdin.readline().strip()
            if not line:
                break
                
            try:
                command = json.loads(line)
            except json.JSONDecodeError:
                response = {"error": "Invalid JSON command"}
                print(json.dumps(response))
                continue
            
            action = command.get("action")
            response = {"action": action, "success": False}
            
            if action == "get_info":
                response["data"] = plugin.get_info()
                response["success"] = True
                
            elif action == "initialize":
                config = command.get("config", {})
                response["success"] = plugin.initialize(config)
                
            elif action == "start":
                response["success"] = plugin.start()
                
            elif action == "stop":
                response["success"] = plugin.stop()
                
            elif action == "get_status":
                response["data"] = plugin.get_status()
                response["success"] = True
                
            elif action == "get_health":
                response["data"] = plugin.get_health()
                response["success"] = True
                
            elif action == "execute_action":
                action_request = command.get("action_request", {})
                response["data"] = plugin.execute_action(action_request)
                response["success"] = True
                
            elif action == "get_action_config":
                response["data"] = plugin.get_action_config()
                response["success"] = True
                
            else:
                response["error"] = f"Unknown action: {action}"
            
            print(json.dumps(response))
            
        except KeyboardInterrupt:
            break
        except Exception as e:
            response = {"error": f"Plugin error: {str(e)}"}
            print(json.dumps(response))


if __name__ == "__main__":
    main() 