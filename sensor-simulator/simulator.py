#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
古代木拱桥结构监测 - 4G DTU 传感器模拟器
模拟10座古代木拱桥的位移、应变、温湿度传感器数据上报
"""

import json
import time
import random
import math
import requests
from datetime import datetime, timedelta
from typing import List, Dict, Any


class BridgeSensorSimulator:
    def __init__(self, api_base_url: str = "http://localhost:8080/api/v1"):
        self.api_base_url = api_base_url
        self.dtu_devices = self._init_dtu_devices()

    def _init_dtu_devices(self) -> List[Dict[str, Any]]:
        bridges = [
            {"bridge_id": 1, "name": "汴水虹桥", "dtu_id": "DTU-BIANSHUI-001"},
            {"bridge_id": 2, "name": "永安桥", "dtu_id": "DTU-YONGAN-002"},
            {"bridge_id": 3, "name": "龙津桥", "dtu_id": "DTU-LONGJIN-003"},
            {"bridge_id": 4, "name": "广济桥", "dtu_id": "DTU-GUANGJI-004"},
            {"bridge_id": 5, "name": "万安桥", "dtu_id": "DTU-WANAN-005"},
            {"bridge_id": 6, "name": "飞虹桥", "dtu_id": "DTU-FEIHONG-006"},
            {"bridge_id": 7, "name": "千乘桥", "dtu_id": "DTU-QIANSHENG-007"},
            {"bridge_id": 8, "name": "安澜桥", "dtu_id": "DTU-ANLAN-008"},
            {"bridge_id": 9, "name": "枫桥", "dtu_id": "DTU-FENGQIAO-009"},
            {"bridge_id": 10, "name": "灞桥", "dtu_id": "DTU-BAQIAO-010"},
        ]

        for bridge in bridges:
            bridge["sensors"] = self._generate_sensors(bridge["bridge_id"])

        return bridges

    def _generate_sensors(self, bridge_id: int) -> List[Dict[str, Any]]:
        sensors = []

        sensor_types = [
            {"type": "displacement", "measurement": "位移", "unit": "mm", "count": 6},
            {"type": "strain", "measurement": "应变", "unit": "μɛ", "count": 8},
            {"type": "temperature", "measurement": "温度", "unit": "°C", "count": 4},
            {"type": "humidity", "measurement": "湿度", "unit": "%RH", "count": 3},
            {"type": "vibration", "measurement": "振动", "unit": "mm/s", "count": 4},
            {"type": "tilt", "measurement": "倾角", "unit": "°", "count": 2},
        ]

        sensor_idx = 1
        for st in sensor_types:
            for i in range(st["count"]):
                sensor_code = f"S{bridge_id:02d}-{st['type'].upper()}-{sensor_idx:03d}"
                sensors.append({
                    "sensor_code": sensor_code,
                    "sensor_type": st["type"],
                    "measurement": st["measurement"],
                    "unit": st["unit"],
                    "base_value": self._get_base_value(st["type"], bridge_id),
                    "amplitude": self._get_amplitude(st["type"]),
                    "noise": self._get_noise(st["type"]),
                    "position": i / st["count"],
                })
                sensor_idx += 1

        return sensors

    def _get_base_value(self, sensor_type: str, bridge_id: int) -> float:
        base_values = {
            "displacement": 5.0 + bridge_id * 0.3,
            "strain": 150.0 + bridge_id * 10,
            "temperature": 22.0,
            "humidity": 60.0,
            "vibration": 0.5,
            "tilt": 0.1,
        }
        return base_values.get(sensor_type, 0.0)

    def _get_amplitude(self, sensor_type: str) -> float:
        amplitudes = {
            "displacement": 2.0,
            "strain": 50.0,
            "temperature": 10.0,
            "humidity": 20.0,
            "vibration": 0.3,
            "tilt": 0.05,
        }
        return amplitudes.get(sensor_type, 0.0)

    def _get_noise(self, sensor_type: str) -> float:
        noises = {
            "displacement": 0.1,
            "strain": 5.0,
            "temperature": 0.3,
            "humidity": 1.0,
            "vibration": 0.05,
            "tilt": 0.005,
        }
        return noises.get(sensor_type, 0.0)

    def _simulate_sensor_value(self, sensor: Dict[str, Any], current_time: datetime) -> float:
        hour_fraction = current_time.hour / 24.0
        day_cycle = math.sin(2 * math.pi * hour_fraction)

        position_effect = math.sin(math.pi * sensor["position"])

        base = sensor["base_value"]
        amplitude = sensor["amplitude"]
        noise = sensor["noise"] * (random.random() - 0.5) * 2

        value = base + amplitude * day_cycle * position_effect + noise

        if sensor["sensor_type"] in ["strain", "displacement", "vibration"]:
            load_factor = 1.0 + 0.3 * math.sin(2 * math.pi * current_time.minute / 60.0)
            value *= load_factor

        return round(value, 4)

    def generate_dtu_payload(self, dtu_device: Dict[str, Any], current_time: datetime) -> Dict[str, Any]:
        sensors_reading = []

        for sensor in dtu_device["sensors"]:
            value = self._simulate_sensor_value(sensor, current_time)
            quality = self._calculate_quality(value, sensor)

            sensors_reading.append({
                "sensor_code": sensor["sensor_code"],
                "value": value,
                "quality": quality,
                "unit": sensor["unit"],
            })

        payload = {
            "dtu_device_id": dtu_device["dtu_id"],
            "timestamp": current_time.isoformat(),
            "sensors": sensors_reading,
            "raw_data": {
                "signal_strength": round(random.uniform(-85, -55), 1),
                "battery_voltage": round(random.uniform(11.5, 13.8), 2),
                "temperature_module": round(random.uniform(20, 35), 1),
            }
        }

        return payload

    def _calculate_quality(self, value: float, sensor: Dict[str, Any]) -> int:
        base = sensor["base_value"]
        amp = sensor["amplitude"]
        ratio = abs(value - base) / amp if amp > 0 else 0

        if ratio < 1.5:
            return 0
        elif ratio < 2.0:
            return 1
        else:
            return 2

    def send_dtu_data(self, payload: Dict[str, Any]) -> bool:
        try:
            url = f"{self.api_base_url}/sensors/dtu-ingest"
            response = requests.post(url, json=payload, timeout=5)

            if response.status_code == 200:
                result = response.json()
                return result.get("status") == "success"
            else:
                print(f"  [ERROR] HTTP {response.status_code}: {response.text}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"  [ERROR] Request failed: {e}")
            return False

    def send_environmental_data(self, bridge_id: int, current_time: datetime) -> bool:
        temperature = 22.0 + 10.0 * math.sin(2 * math.pi * current_time.hour / 24.0)
        temperature += random.uniform(-0.5, 0.5)

        humidity = 60.0 + 20.0 * math.cos(2 * math.pi * current_time.hour / 24.0)
        humidity += random.uniform(-2, 2)
        humidity = max(20, min(95, humidity))

        wind_speed = random.uniform(0, 5)
        wind_direction = random.uniform(0, 360)
        rainfall = max(0, random.uniform(-0.5, 2))

        env_data = {
            "bridge_id": bridge_id,
            "timestamp": current_time.isoformat(),
            "temperature": round(temperature, 2),
            "humidity": round(humidity, 2),
            "wind_speed": round(wind_speed, 2),
            "wind_direction": round(wind_direction, 1),
            "rainfall": round(rainfall, 2),
        }

        print(f"  Environmental: {env_data['temperature']}°C, {env_data['humidity']}%RH")
        return True

    def run_simulation(self, interval_seconds: int = 3600, duration_hours: int = 24):
        print("=" * 60)
        print("古代木拱桥结构监测 - 4G DTU 传感器模拟器启动")
        print("=" * 60)
        print(f"API地址: {self.api_base_url}")
        print(f"上报间隔: {interval_seconds}秒 ({interval_seconds//60}分钟)")
        print(f"模拟桥梁数: {len(self.dtu_devices)}")
        print(f"传感器总数: {sum(len(b['sensors']) for b in self.dtu_devices)}")
        print("=" * 60)

        current_time = datetime.now().replace(minute=0, second=0, microsecond=0)
        total_updates = duration_hours * 3600 // interval_seconds

        for i in range(total_updates):
            print(f"\n[第 {i+1}/{total_updates} 轮] 时间: {current_time.strftime('%Y-%m-%d %H:%M:%S')}")
            print("-" * 60)

            for dtu in self.dtu_devices:
                print(f"\n[{dtu['name']}] ({dtu['dtu_id']})")

                payload = self.generate_dtu_payload(dtu, current_time)

                success = self.send_dtu_data(payload)
                status = "✓" if success else "✗"
                print(f"  {status} 上报传感器数据: {len(payload['sensors'])} 个测点")

                self.send_environmental_data(dtu["bridge_id"], current_time)

            current_time += timedelta(seconds=interval_seconds)

            if i < total_updates - 1:
                time.sleep(1)

        print("\n" + "=" * 60)
        print("模拟完成")
        print("=" * 60)

    def run_realtime_simulation(self, interval_seconds: int = 60):
        print("=" * 60)
        print("古代木拱桥结构监测 - 实时模拟模式")
        print("=" * 60)
        print("按 Ctrl+C 停止模拟")
        print()

        try:
            while True:
                current_time = datetime.now()
                print(f"[{current_time.strftime('%Y-%m-%d %H:%M:%S')}] 正在上报数据...")

                for dtu in self.dtu_devices:
                    payload = self.generate_dtu_payload(dtu, current_time)
                    success = self.send_dtu_data(payload)
                    if success:
                        print(f"  ✓ {dtu['name']}: {len(payload['sensors'])} 个测点")
                    else:
                        print(f"  ✗ {dtu['name']}: 上报失败")

                print(f"  等待 {interval_seconds} 秒后下一次上报...")
                time.sleep(interval_seconds)

        except KeyboardInterrupt:
            print("\n\n模拟已停止")


def generate_historical_data(api_url: str, days: int = 30):
    print(f"生成过去 {days} 天的历史数据...")

    simulator = BridgeSensorSimulator(api_url)

    end_time = datetime.now().replace(minute=0, second=0, microsecond=0)
    start_time = end_time - timedelta(days=days)

    current_time = start_time
    total_hours = days * 24
    count = 0

    while current_time <= end_time:
        for dtu in simulator.dtu_devices:
            payload = simulator.generate_dtu_payload(dtu, current_time)
            simulator.send_dtu_data(payload)

        count += 1
        if count % 24 == 0:
            print(f"  已生成 {count}/{total_hours} 小时数据 ({current_time.strftime('%Y-%m-%d')})")

        current_time += timedelta(hours=1)

    print(f"历史数据生成完成: {count} 小时")


def main():
    import argparse

    parser = argparse.ArgumentParser(description="古代木拱桥4G DTU传感器模拟器")
    parser.add_argument("--api-url", default="http://localhost:8080/api/v1",
                        help="后端API地址")
    parser.add_argument("--mode", choices=["realtime", "batch", "historical"],
                        default="realtime", help="运行模式")
    parser.add_argument("--interval", type=int, default=60,
                        help="上报间隔（秒），实时模式使用")
    parser.add_argument("--duration", type=int, default=24,
                        help="模拟时长（小时），批量模式使用")
    parser.add_argument("--days", type=int, default=30,
                        help="历史数据天数，历史模式使用")

    args = parser.parse_args()

    simulator = BridgeSensorSimulator(args.api_url)

    if args.mode == "realtime":
        simulator.run_realtime_simulation(args.interval)
    elif args.mode == "batch":
        simulator.run_simulation(interval_seconds=3600, duration_hours=args.duration)
    elif args.mode == "historical":
        generate_historical_data(args.api_url, args.days)


if __name__ == "__main__":
    main()
