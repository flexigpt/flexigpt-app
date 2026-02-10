import { type ReactNode, useState } from 'react';

import { FiFilePlus, FiMessageSquare, FiSettings, FiSliders, FiTool } from 'react-icons/fi';

import { Link } from 'react-router';

import { TitleBar } from '@/components/title_bar';

interface SidebarProps {
	children: ReactNode;
}

export function Sidebar({ children }: SidebarProps) {
	const [isDrawerOpen, setDrawerOpen] = useState(false);
	const toggle = () => {
		setDrawerOpen(!isDrawerOpen);
	};

	return (
		<div className="drawer lg:drawer-open h-screen">
			<input
				id="my-drawer"
				type="checkbox"
				className="drawer-toggle lg:hidden"
				checked={isDrawerOpen}
				onChange={toggle}
				spellCheck="false"
			/>
			<div className="drawer-content flex min-h-0 flex-col overflow-hidden">
				<TitleBar
					onToggleDrawer={() => {
						setDrawerOpen(o => !o);
					}}
				/>
				<div className="min-h-0 flex-1 overflow-hidden">{children}</div>
			</div>
			<div className="drawer-side z-10">
				<label htmlFor="my-drawer" className="drawer-overlay"></label>
				<ul className="menu bg-base-300 text-base-content ms-0 h-full w-12 justify-between ps-0">
					<div className="mt-8 flex-col p-0">
						<li className="mt-4" title="Chats" onClick={toggle} aria-label="Chats">
							<Link to="/chats/" className="flex h-12 w-12 items-center justify-center rounded-full p-0">
								<FiMessageSquare size={24} />
							</Link>
						</li>
					</div>
					<div className="mb-8 flex-col p-0">
						<li className="mt-4" title="Tools" onClick={toggle} aria-label="Tools">
							<Link to="/tools/" className="flex h-12 w-12 items-center justify-center rounded-full p-0">
								<FiTool size={24} />
							</Link>
						</li>

						<li className="mt-4" title="Prompts" onClick={toggle} aria-label="Prompts">
							<Link to="/prompts/" className="flex h-12 w-12 items-center justify-center rounded-full p-0">
								<FiFilePlus size={24} />
							</Link>
						</li>

						<li className="mt-4" title="Model Presets" onClick={toggle} aria-label="Model Presets">
							<Link to="/modelpresets/" className="flex h-12 w-12 items-center justify-center rounded-full p-0">
								<FiSliders size={24} />
							</Link>
						</li>
						<li className="mt-4" title="Settings" onClick={toggle} aria-label="Settings">
							<Link to="/settings/" className="flex h-12 w-12 items-center justify-center rounded-full p-0">
								<FiSettings size={24} />
							</Link>
						</li>
					</div>
				</ul>
			</div>
		</div>
	);
}
