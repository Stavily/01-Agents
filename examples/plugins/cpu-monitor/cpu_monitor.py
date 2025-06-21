#!/usr/bin/env python3
"""
CPU Monitor Trigger Plugin for Stavily Sensor Agents

This plugin monitors CPU usage and triggers alerts when threshold is exceeded.
"""

import json
import os
import sys
import time
import psutil
import logging
from datetime import datetime
from typing import Dict, Any, Optional

class CPUMonitorPlugin:
    """CPU Monitor trigger plugin implementation."""
    
    def __init__(self):
        self.config = {}
        self.threshold = 80.0
        self.interval = 30
        self.running = False
        self.status = "stopped"
        self.start_time = None
        self.logger = self._setup_logging()
    
    def _setup_logging(self) -> logging.Logger:
        """Setup plugin logging."""
        logger = logging.getLogger('cpu_monitor')
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
            "id": "cpu-monitor",
            "name": "CPU Monitor",
            "description": "Monitors CPU usage and triggers alerts when threshold is exceeded",
            "version": "1.0.0",
            "author": "Stavily Team",
            "license": "MIT",
            "type": "trigger",
            "tags": ["system", "monitoring", "cpu"],
            "categories": ["system-monitoring"],
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat()
        }
    
    def initialize(self, config: Dict[str, Any]) -> bool:
        """Initialize the plugin with configuration."""
        try:
            self.config = config
            self.threshold = float(config.get("threshold", 80.0))
            self.interval = int(config.get("interval", 30))
            
            # Validate configuration
            if not (0 <= self.threshold <= 100):
                raise ValueError("Threshold must be between 0 and 100")
            if self.interval < 1:
                raise ValueError("Interval must be at least 1 second")
                
            self.status = "initialized"
            self.logger.info(f"CPU Monitor initialized with threshold={self.threshold}%, interval={self.interval}s")
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
            self.logger.info("CPU Monitor plugin started")
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
            self.logger.info("CPU Monitor plugin stopped")
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
                "current_cpu_usage": self._get_cpu_usage(),
                "threshold": self.threshold
            }
        }
        
        if self.start_time:
            uptime = (datetime.now() - self.start_time).total_seconds()
            health["uptime"] = uptime
            
        return health
    
    def detect_triggers(self) -> Optional[Dict[str, Any]]:
        """Detect and return trigger events."""
        if not self.running:
            return None
            
        try:
            cpu_usage = self._get_cpu_usage()
            
            if cpu_usage > self.threshold:
                event = {
                    "id": f"cpu-high-{int(time.time())}",
                    "type": "cpu.high",
                    "source": "cpu-monitor",
                    "timestamp": datetime.now().isoformat(),
                    "data": {
                        "cpu_usage": cpu_usage,
                        "threshold": self.threshold,
                        "cpu_count": psutil.cpu_count(),
                        "load_average": os.getloadavg() if hasattr(os, 'getloadavg') else None
                    },
                    "metadata": {
                        "plugin_id": "cpu-monitor",
                        "plugin_version": "1.0.0",
                        "hostname": os.uname().nodename
                    },
                    "tags": ["system", "cpu", "alert"],
                    "severity": "high" if cpu_usage > 90 else "medium"
                }
                
                self.logger.warning(f"CPU usage trigger detected: {cpu_usage:.1f}% > {self.threshold}%")
                return event
                
            return None
            
        except Exception as e:
            self.logger.error(f"Error detecting triggers: {e}")
            return None
    
    def get_trigger_config(self) -> Dict[str, Any]:
        """Return the trigger configuration schema."""
        return {
            "schema": {
                "threshold": {
                    "type": "number",
                    "description": "CPU usage threshold percentage (0-100)",
                    "default": 80.0,
                    "required": False,
                    "minimum": 0.0,
                    "maximum": 100.0
                },
                "interval": {
                    "type": "integer",
                    "description": "Monitoring interval in seconds",
                    "default": 30,
                    "required": False,
                    "minimum": 1,
                    "examples": [30, 60, 300]
                }
            },
            "required": [],
            "examples": [
                {
                    "threshold": 85.0,
                    "interval": 30
                },
                {
                    "threshold": 90.0,
                    "interval": 60
                }
            ],
            "description": "CPU monitoring configuration"
        }
    
    def _get_cpu_usage(self) -> float:
        """Get current CPU usage percentage."""
        return psutil.cpu_percent(interval=1)


def main():
    """Main plugin entry point."""
    plugin = CPUMonitorPlugin()
    
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
                
            elif action == "detect_triggers":
                trigger_event = plugin.detect_triggers()
                response["data"] = trigger_event
                response["success"] = True
                
            elif action == "get_trigger_config":
                response["data"] = plugin.get_trigger_config()
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