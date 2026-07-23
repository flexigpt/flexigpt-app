import type { ReactNode } from 'react';
import { useEffect, useRef, useState } from 'react';

import {
	FiBookOpen,
	FiFilePlus,
	FiLayers,
	FiMessageSquare,
	FiServer,
	FiSettings,
	FiSliders,
	FiTool,
} from 'react-icons/fi';

import { Link } from 'react-router';

import { installDropGuard } from '@/lib/dropblocker';

import { TitleBar } from '@/components/title_bar';

interface SidebarProps {
	children: ReactNode;
}

export function Sidebar({ children }: SidebarProps) {
	const [drawerOpen, setDrawerOpen] = useState(false);
	const toggle = () => {
		setDrawerOpen(open => !open);
	};

	const dropRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		if (!dropRef.current) {
			return;
		}

		// Ideally installDropGuard returns a cleanup function:
		const cleanup = installDropGuard(dropRef.current);
		return cleanup;
	}, []);

	return (
		<div className="drawer lg:drawer-open h-screen">
			<input
				id="my-drawer"
				type="checkbox"
				className="drawer-toggle lg:hidden"
				checked={drawerOpen}
				onChange={toggle}
				spellCheck="false"
			/>
			<div className="drawer-content flex min-h-0 flex-col overflow-hidden">
				<TitleBar
					onToggleDrawer={() => {
						setDrawerOpen(o => !o);
					}}
				/>
				<div ref={dropRef} className="min-h-0 flex-1 overflow-hidden">
					{children}
				</div>
			</div>
			<div className="drawer-side z-10">
				<label htmlFor="my-drawer" aria-label="Close navigation drawer" />
				<ul className="menu bg-base-300 text-base-content ms-0 h-full w-12 justify-between ps-0">
					<div className="mt-8 flex-col p-0">
						<li className="mt-4">
							<Link
								to="/chats/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Chats"
								title="Chats"
							>
								<FiMessageSquare size={24} />
							</Link>
						</li>
					</div>
					<div className="mb-8 flex-col p-0">
						<li className="mt-4">
							<Link
								to="/assistantpresets/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Assistant Presets"
								title="Assistant Presets"
							>
								<FiLayers size={24} />
							</Link>
						</li>
						{/* <li className="mt-4">
							<Link
								to="/workspaces/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Workspaces"
								title="Workspaces"
							>
								<FiFolder size={24} />
							</Link>
						</li> */}
						<li className="mt-4">
							<Link
								to="/mcpservers/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="MCP Servers"
								title="MCP Servers"
							>
								<FiServer size={24} />
							</Link>
						</li>
						<li className="mt-4">
							<Link
								to="/tools/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Tools"
								title="Tools"
							>
								<FiTool size={24} />
							</Link>
						</li>
						<li className="mt-4">
							<Link
								to="/skills/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Skills"
								title="Skills"
							>
								<FiFilePlus size={24} />
							</Link>
						</li>
						<li className="mt-4">
							<Link
								to="/modelpresets/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Model Presets"
								title="Model Presets"
							>
								<FiSliders size={24} />
							</Link>
						</li>

						<li className="mt-4">
							<Link
								to="/settings/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Settings"
								title="Settings"
							>
								<FiSettings size={24} />
							</Link>
						</li>

						<li className="mt-4">
							<Link
								to="/docs/"
								className="flex size-12 items-center justify-center rounded-full p-0"
								onClick={toggle}
								aria-label="Docs"
								title="Docs"
							>
								<FiBookOpen size={24} />
							</Link>
						</li>
					</div>
				</ul>
			</div>
		</div>
	);
}
