i have this attached workflow and its runtime system. the hlds are attached.

You need to analyze and report things related to my "Runtime" and associated "setup" things. artifact base is also attached but assume that it is ok as is.

I am now going to start sessionv2 migration to sessionv3.

Note that i will be in new branch and this will be breaking migration. i.e old sessions etc will stay as is but will not migrate them anywhere.
I will change base dir to sessionv3 (I have already done this) and then new sessions will start there. workflowv1 is just introduced so no break there.

You need to:

- Identify any BLOCKERS and tasks to do to complete this migration of BACKEND only as of now.
- identify redundancy, or duplication of code. arch boundary violations etc too.
- ignore anything related to tests, validations (any and all validations), sessionv2 cleanup activity, frontend integration
- gaps wrt incompleteness for replacing sessionv2 still needs to be identified for above but these gaps need to be functional capability fully missing ones only if these need ot be done before. Else, bugs to fix can be documented in the migration doc itself which will be solved along with the backend migration.
- note that current v2 supports single completion flow and associated things i.e base llm completion loop things. in the new migrated thing this thing needs to be FULLY supported i.e all capabilities need to be fully available from current v2 to sessionv3. missing completeness wrt additional workflow things is fine and can be ignored as of now.

once you do this, explain the PENDING items only i.e that needs to be still completed in functional detail i.e what functionality is missing.
then give proper contextual unified diffs to both or any hld as required to rectify and mention the above implementation status.
Then give a NEW migration HLD as a full file that will detail all things related to it. As noted above, unless and until FULL functionality is missing, you do not need to update old HLDs and can add things to migration HLD directly. Maintain the current HLDs structure of mentioning goals, workflows and last functional task format so that this can be a working document.

This is NOT a code task. this is a analysis and work items setting tasks.
No phases etc, just functional things i.e functional task and subtask split and status.
Note that artifact hld should stay artifact store and retrieval and usage focused only, it should not really deal with anything on HOW it is used in runtime anything as it is not its concern. Maintain arch separation and separation of concerns appropriately.

Make sure you do document all arch and deficiencies etc analysis. same with duplications. These should be in enough detail so that some agent can act on it without inspecting a lot again and invalidating some findings you have.

After the diffs, if there are pending items, give a single prompt to complete the migration. Explicitly mention in prompt to ignore tests and frontend and other things i mentioned. Explicitly mention to remove duplications, domain owning domain things and runtime owning runtime and not depending on artifact and session/workflow remaining as orchestration. Redundancy removal, reducing validations should also be mentioned properly.
If you think anything will be pending even after the prompts proper implementation, state here separately (not as a prompt, just items) what will be remaining.

Make appropriate space in the hld where the status of completed items can be marked in the hld (i.e current status should show not implemented or appropriate thing and then the implementer will mark it complete as such). tell in prompts for implementer to do so i.e update statuses in hld.

again, tests, removal of v2, frontend integration is not part of the implementation cycle. say explicitly in hld (if not already there)and in prompts too that this is a WIP and no versions etc should be bumped anywhere. Also mention that this need to be completed properly and a large single shot outputs are expected and not some small things should also be mentioned. Note that the direct tool capability in v2 is redundant was analyzed and documented already. do not repeat that.
In prompt mention that we dont use hardcoded strings, we need const declarations like now. proper enums should be used everywhere and error strings are to be lowercase. also mention that code delivery should be as per code delivery instructions (which i will attach separately). no tools will be attached. only unified diff outputs in the specified format needs to be given.

lets do a deep and truthful analysis and give concrete output now.

make sure to capture all findings properly and proper places in the HLDs.
